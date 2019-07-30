package packet

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/klogr"
	packetconfig "pharmer.dev/pharmer/apis/v1alpha1/packet"
	"pharmer.dev/pharmer/cloud"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/controller/machine"
	"sigs.k8s.io/cluster-api/pkg/kubeadm"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, func(cm *ClusterManager, m manager.Manager) error {
		actuator := NewMachineActuator(MachineActuatorParams{
			EventRecorder: m.GetEventRecorderFor(Recorder),
			Client:        m.GetClient(),
			cm:            cm,
		})
		return machine.AddWithActuator(m, actuator)
	})
}

type DOClientKubeadm interface {
	TokenCreate(params kubeadm.TokenCreateParams) (string, error)
}

type MachineActuator struct {
	cm            *ClusterManager
	client        client.Client
	kubeadm       DOClientKubeadm
	eventRecorder record.EventRecorder
}

type MachineActuatorParams struct {
	cm             *ClusterManager
	Kubeadm        DOClientKubeadm
	Client         client.Client
	CloudConnector *cloudConnector
	EventRecorder  record.EventRecorder
}

func NewMachineActuator(params MachineActuatorParams) *MachineActuator {
	params.cm.Logger = klogr.New().WithName("[machine-actuator]").
		WithValues("cluster-name", params.cm.Cluster.Name)
	return &MachineActuator{
		cm:            params.cm,
		client:        params.Client,
		kubeadm:       getKubeadm(params),
		eventRecorder: params.EventRecorder,
	}
}

func (packet *MachineActuator) Create(_ context.Context, cluster *clusterapi.Cluster, machine *clusterapi.Machine) error {
	log := packet.cm.Logger.WithValues("machine-name", machine.Name)
	log.Info("call for creating machine")

	machineConfig, err := packetconfig.MachineConfigFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		log.Error(err, "error decoding provider config for machine")
		return err
	}

	if err := packet.validateMachine(machineConfig); err != nil {
		log.Error(err, "failed to validate machine config")
		return err
	}

	exists, err := packet.Exists(context.Background(), cluster, machine)
	if err != nil {
		log.Error(err, "failed to check existence of machine")
		return err
	}

	if exists {
		log.Info("Skipped creating a machine that already exists")
	} else {
		log.Info("vm not found, creating vm for machine")

		token, err := packet.getKubeadmToken()
		if err != nil {
			log.Error(err, "failed to generate kubadm token")
			return err
		}

		script, err := cloud.RenderStartupScript(packet.cm, machine, token, customTemplate) // ClusterManager is needed here
		if err != nil {
			log.Error(err, "failed to render statup script")
			return err
		}

		_, err = packet.cm.conn.CreateInstance(machine, script)
		if err != nil {
			log.Error(err, "failed to create instance")
			return err
		}
	}

	// update machine provider status
	err = packet.updateMachineStatus(machine)
	if err != nil {
		log.Error(err, "failed to update machine status")
		return errors.Wrap(err, "failed to update machine status")
	}

	log.Info("successfully created machine")
	return nil
}

func (packet *MachineActuator) updateMachineStatus(machine *clusterapi.Machine) error {
	log := packet.cm.Logger
	vm, err := packet.cm.conn.instanceIfExists(machine)
	if err != nil {
		log.Error(err, "failed to check machine existence")
		return err
	}

	status, err := packetconfig.MachineStatusFromProviderStatus(machine.Status.ProviderStatus)
	if err != nil {
		log.Error(err, "failed to get machine status")
		return err
	}

	status.InstanceID = vm.ID

	raw, err := packetconfig.EncodeMachineStatus(status)
	if err != nil {
		log.Error(err, "failed to encode provider status")
		return err
	}
	machine.Status.ProviderStatus = raw

	if err := packet.client.Status().Update(context.Background(), machine); err != nil {
		log.Error(err, "failed to update provider status")
		return err
	}

	return nil
}

func (packet *MachineActuator) validateMachine(providerConfig *packetconfig.PacketMachineProviderSpec) error {
	if len(providerConfig.Plan) == 0 {
		return errors.New("plan must be provided")
	}
	return nil
}

func (packet *MachineActuator) getKubeadmToken() (string, error) {
	tokenParams := kubeadm.TokenCreateParams{
		TTL: time.Duration(30) * time.Minute,
	}

	token, err := packet.kubeadm.TokenCreate(tokenParams)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(token), nil
}

func machineProviderFromProviderSpec(providerSpec clusterapi.ProviderSpec) (*packetconfig.PacketMachineProviderSpec, error) {
	var config packetconfig.PacketMachineProviderSpec
	if err := yaml.Unmarshal(providerSpec.Value.Raw, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func (packet *MachineActuator) Delete(_ context.Context, cluster *clusterapi.Cluster, machine *clusterapi.Machine) error {
	log := packet.cm.Logger.WithValues("machine-name", machine.Name)
	log.Info("call for deleting machine")
	var err error

	instance, err := packet.cm.conn.instanceIfExists(machine)
	if err != nil {
		log.Error(err, "instance doesn't exist")
	}
	if instance == nil {
		log.Info("Skipped deleting a VM that is already deleted")
		return nil
	}
	serverId := fmt.Sprintf("packet://%v", instance.ID)

	if err = packet.cm.conn.DeleteInstanceByProviderID(serverId); err != nil {
		log.Error(err, "error deleting instance")
	}

	packet.eventRecorder.Eventf(machine, corev1.EventTypeNormal, "Deleted", "Deleted Machine %v", machine.Name)

	return nil
}

func (packet *MachineActuator) Update(_ context.Context, cluster *clusterapi.Cluster, machine *clusterapi.Machine) error {
	log := packet.cm.Logger.WithValues("machine-name", machine.Name)
	log.Info("updating machine")

	var err error
	goalConfig, err := machineProviderFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		log.Error(err, "failed to decode machine provider spec")
		return err
	}

	if err := packet.validateMachine(goalConfig); err != nil {
		log.Error(err, "failed to validate machine")
		return err
	}

	exists, err := packet.Exists(context.Background(), cluster, machine)
	if err != nil {
		log.Error(err, "failed to check machine existence")
		return err
	}

	if !exists {
		log.Info("vm not found, creating vm for machine")
		return packet.Create(context.Background(), cluster, machine)
	}

	if err := packet.updateMachineStatus(machine); err != nil {
		log.Error(err, "failed to update machine status")
		return err
	}

	log.Info("Successfully updated machine")
	return nil
}

func (packet *MachineActuator) Exists(ctx context.Context, cluster *clusterapi.Cluster, machine *clusterapi.Machine) (bool, error) {
	log := packet.cm.Logger.WithValues("machine-name", machine.Name)
	log.Info("call for checking machine existence")
	var err error

	i, err := packet.cm.conn.instanceIfExists(machine)
	if err != nil {
		log.Error(err, "failed to check machine existence")
		return false, nil
	}

	return i != nil, nil
}

func getKubeadm(params MachineActuatorParams) DOClientKubeadm {
	if params.Kubeadm == nil {
		return kubeadm.New()
	}
	return params.Kubeadm
}
