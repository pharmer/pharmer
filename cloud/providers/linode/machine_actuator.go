package linode

import (
	"context"
	"fmt"
	"strings"
	"time"

	linodeconfig "github.com/pharmer/pharmer/apis/v1beta1/linode"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/controller/machine"
	"sigs.k8s.io/cluster-api/pkg/kubeadm"
	"sigs.k8s.io/cluster-api/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, func(cm *ClusterManager, m manager.Manager) error {
		actuator := NewMachineActuator(MachineActuatorParams{
			EventRecorder: m.GetEventRecorderFor(Recorder),
			Client:        m.GetClient(),
			Scheme:        m.GetScheme(),
			cm:            cm,
		})
		return machine.AddWithActuator(m, actuator)
	})
}

type LinodeClientKubeadm interface {
	TokenCreate(params kubeadm.TokenCreateParams) (string, error)
}

type MachineActuator struct {
	cm            *ClusterManager
	client        client.Client
	kubeadm       LinodeClientKubeadm
	eventRecorder record.EventRecorder
	scheme        *runtime.Scheme
}

type MachineActuatorParams struct {
	Kubeadm       LinodeClientKubeadm
	Client        client.Client
	cm            *ClusterManager
	EventRecorder record.EventRecorder
	Scheme        *runtime.Scheme
}

func NewMachineActuator(params MachineActuatorParams) *MachineActuator {
	return &MachineActuator{
		cm:            params.cm,
		client:        params.Client,
		kubeadm:       getKubeadm(params),
		eventRecorder: params.EventRecorder,
		scheme:        params.Scheme,
		//owner:         params.Owner,
		//machineSetupConfigGetter: MachineSetup(params.Ctx),
	}
}

func (li *MachineActuator) Create(ctx context.Context, cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	log := li.cm.Logger

	log.Info("creating machine", "machine-name", machine.Name)

	machineConfig, err := linodeconfig.MachineConfigFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		return errors.Wrapf(err, "error decoding provider config for macine %s", machine.Name)
	}

	if err := li.validateMachine(machineConfig); err != nil {
		return errors.Wrapf(err, "failed to valide machine config for machien %s", machine.Name)
	}

	exists, err := li.Exists(context.Background(), cluster, machine)
	if err != nil {
		return errors.Wrapf(err, "failed to check existance of machine %s", machine.Name)
	}

	if exists {
		log.Info("Skipped creating a machine that already exists.")
	} else {
		log.Info("vm not found, creating vm for machine.", "machine-name", machine.Name)

		token, err := li.getKubeadmToken()
		if err != nil {
			log.Error(err, "failed to generate kubeadm token")
			return errors.Wrap(err, "failed to generate kubeadm token")
		}

		script, err := cloud.RenderStartupScript(li.cm, machine, token, customTemplate)
		if err != nil {
			log.Error(err, "failed to render st script")
			return err
		}

		server, err := li.cm.conn.CreateInstance(machine, script)
		if err != nil {
			log.Error(err, "failed to create instance")
			return errors.Wrap(err, "failed to create instance")
		}

		if util.IsControlPlaneMachine(machine) {
			if err = li.cm.conn.addNodeToBalancer(li.cm.conn.namer.LoadBalancerName(), machine.Name, server.PrivateIP); err != nil {
				return errors.Wrap(err, "failed to add machine to load balancer")
			}
		}
	}

	// update machine provider status
	err = li.updateMachineStatus(machine)
	if err != nil {
		return errors.Wrap(err, "failed to update machine status")
	}

	log.Info("successfully created machine", "machine-name", machine.Name)
	return nil
}

func (li *MachineActuator) validateMachine(providerConfig *linodeconfig.LinodeMachineProviderSpec) error {
	if len(providerConfig.Image) == 0 {
		return errors.New("image slug must be provided")
	}
	if len(providerConfig.Region) == 0 {
		return errors.New("region must be provided")
	}
	if len(providerConfig.Type) == 0 {
		return errors.New("type must be provided")
	}

	return nil
}

func (li *MachineActuator) getKubeadmToken() (string, error) {
	tokenParams := kubeadm.TokenCreateParams{
		TTL: time.Duration(30) * time.Minute,
	}

	token, err := li.kubeadm.TokenCreate(tokenParams)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(token), nil
}

func (li *MachineActuator) Delete(_ context.Context, cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	log := li.cm.Logger

	log.Info("deleting machine", "machinej-name", machine.Name)

	var err error

	instance, err := li.cm.conn.instanceIfExists(machine)
	if err != nil {
		log.Error(err, "skipping error")
	}
	if instance == nil {
		log.Info("Skipped deleting a VM that is already deleted")
		return nil
	}
	instanceID := fmt.Sprintf("linode://%v", instance.ID)

	if err = li.cm.conn.DeleteInstanceByProviderID(instanceID); err != nil {
		log.Error(err, "error deleting instance", "instance-id", instanceID)
	}

	li.eventRecorder.Eventf(machine, corev1.EventTypeNormal, "Deleted", "Deleted Machine %v", machine.Name)

	log.Info("successfully deleted machine", "machine-name", machine.Name)
	return nil
}

func (li *MachineActuator) Update(_ context.Context, cluster *clusterv1.Cluster, goalMachine *clusterv1.Machine) error {
	log := li.cm.Logger

	log.Info("updating machine", "machine-name", goalMachine.Name)
	var err error

	goalConfig, err := linodeconfig.MachineConfigFromProviderSpec(goalMachine.Spec.ProviderSpec)
	if err != nil {
		return errors.Wrap(err, "failed to decode provider spec")
	}

	if err := li.validateMachine(goalConfig); err != nil {
		return errors.Wrap(err, "failed to validate machine")
	}

	exists, err := li.Exists(context.Background(), cluster, goalMachine)
	if err != nil {
		return errors.Wrapf(err, "failed to check existance of machine %s", goalMachine.Name)
	}

	if !exists {
		log.Info("vm not found, creating vm for machine.", "machine-name", goalMachine.Name)
		return li.Create(context.Background(), cluster, goalMachine)
	}

	if err := li.updateMachineStatus(goalMachine); err != nil {
		return errors.Wrap(err, "failed to update machine status")
	}

	log.Info("Successfully updated machine", "machine-name", goalMachine.Name)
	return nil
}

func (li *MachineActuator) Exists(ctx context.Context, cluster *clusterv1.Cluster, machine *clusterv1.Machine) (bool, error) {
	log := li.cm.Logger

	log.Info("checking existance of machine", "machine-name", machine.Name)
	var err error

	i, err := li.cm.conn.instanceIfExists(machine)
	if err != nil {
		log.Error(err, "error checking machine existence")
		return false, nil
	}

	return i != nil, nil
}

func getKubeadm(params MachineActuatorParams) LinodeClientKubeadm {
	if params.Kubeadm == nil {
		return kubeadm.New()
	}
	return params.Kubeadm
}

func (li *MachineActuator) updateMachineStatus(machine *clusterv1.Machine) error {
	vm, err := li.cm.conn.instanceIfExists(machine)
	if err != nil {
		return errors.Wrapf(err, "failed to check existance of machine %s", machine.Name)
	}

	status, err := linodeconfig.MachineStatusFromProviderStatus(machine.Status.ProviderStatus)
	if err != nil {
		return errors.Wrapf(err, "failed to decode provider status of machine %s", machine.Name)
	}

	status.InstanceID = vm.ID
	status.InstanceStatus = string(vm.Status)

	raw, err := linodeconfig.EncodeMachineStatus(status)
	if err != nil {
		return errors.Wrapf(err, "failed to encode provider status for machine %q", machine.Name)
	}
	machine.Status.ProviderStatus = raw

	if err := li.client.Status().Update(context.Background(), machine); err != nil {
		return errors.Wrapf(err, "failed to update provider status for machine %s", machine.Name)
	}

	return nil
}
