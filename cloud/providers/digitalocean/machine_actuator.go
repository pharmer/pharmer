package digitalocean

import (
	"context"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/klogr"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/controller/machine"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	//	"k8s.io/client-go/kubernetes"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	//kubeadmconsts "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	doCapi "github.com/pharmer/pharmer/apis/v1beta1/digitalocean"
	"sigs.k8s.io/cluster-api/pkg/kubeadm"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	client        client.Client
	kubeadm       DOClientKubeadm
	eventRecorder record.EventRecorder
	cm            *ClusterManager
	logr.Logger
}

type MachineActuatorParams struct {
	Kubeadm       DOClientKubeadm
	Client        client.Client
	EventRecorder record.EventRecorder
	cm            *ClusterManager
}

func NewMachineActuator(params MachineActuatorParams) *MachineActuator {
	params.cm.Logger = params.cm.Logger.WithName("[machine-actuator]")
	return &MachineActuator{
		client:        params.Client,
		kubeadm:       getKubeadm(params),
		eventRecorder: params.EventRecorder,
		cm:            params.cm,
		Logger: klogr.New().WithName("[machine-actuator]").
			WithValues("cluster-name", params.cm.Cluster.Name),
	}
}

func (do *MachineActuator) Create(_ context.Context, cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	log := do.Logger.WithValues("machine-name", machine.Name)
	log.Info("call for creating machine")

	machineConfig, err := doCapi.MachineConfigFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		log.Error(err, "failed to decode provider config for machine")
		return err
	}

	if err := do.validateMachine(machineConfig); err != nil {
		log.Error(err, "failed to validate machine config")
		return err
	}

	exists, err := do.Exists(context.Background(), cluster, machine)
	if err != nil {
		log.Error(err, "failed to check existance of machine")
		return err
	}

	if exists {
		log.Info("Machine exists, skipping machine creation")
	} else {
		log.Info("Droplet not found, creating droplet for machine")

		token, err := do.getKubeadmToken()
		if err != nil {
			log.Error(err, "failed to generate kubeadm token")
			return err
		}

		script, err := cloud.RenderStartupScript(do.cm, machine, token, customTemplate)
		if err != nil {
			log.Error(err, "failed to render start up script")
			return err
		}

		if err := do.cm.conn.CreateInstance(do.cm.conn.Cluster, machine, script); err != nil {
			log.Error(err, "failed to create machine instance")
			return err
		}
	}

	// update machine provider status
	err = do.updateMachineStatus(machine)
	if err != nil {
		log.Error(err, "failed to update machine status")
		return err
	}

	log.Info("Successfully created machine")
	return nil
}

func (do *MachineActuator) updateMachineStatus(machine *clusterv1.Machine) error {
	log := do.Logger
	droplet, err := do.cm.conn.instanceIfExists(machine)
	if err != nil {
		log.Error(err, "Error checking existance for machine")
		return err
	}

	status, err := doCapi.MachineStatusFromProviderStatus(machine.Status.ProviderStatus)
	if err != nil {
		log.Error(err, "Error getting machine status for machine")
		return err
	}

	status.InstanceID = droplet.ID
	status.InstanceStatus = droplet.Status

	raw, err := doCapi.EncodeMachineStatus(status)
	if err != nil {
		log.Error(err, "Error encoding machine status for machine")
	}

	machine.Status.ProviderStatus = raw

	if err := do.client.Status().Update(context.Background(), machine); err != nil {
		log.Error(err, "Error updating status for machine")
		return err
	}

	return nil
}

func (do *MachineActuator) validateMachine(providerConfig *doCapi.DigitalOceanMachineProviderSpec) error {
	if len(providerConfig.Image) == 0 {
		return errors.New("image slug must be provided")
	}
	if len(providerConfig.Region) == 0 {
		return errors.New("region must be provided")
	}
	if len(providerConfig.Size) == 0 {
		return errors.New("size must be provided")
	}

	return nil
}

func (do *MachineActuator) getKubeadmToken() (string, error) {
	tokenParams := kubeadm.TokenCreateParams{
		TTL: time.Duration(30) * time.Minute,
	}

	token, err := do.kubeadm.TokenCreate(tokenParams)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(token), nil
}

func (do *MachineActuator) Delete(_ context.Context, cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	log := do.Logger.WithValues("machine-name", machine.Name)
	log.Info("call for deleting machine")
	var err error

	instance, err := do.cm.conn.instanceIfExists(machine)
	if err != nil {
		log.Error(err, "error checking machine existance")
	}
	if instance == nil {
		log.Info("Skipped deleting a VM that is already deleted")
		return nil
	}
	dropletID := fmt.Sprintf("digitalocean://%v", instance.ID)

	if err = do.cm.conn.DeleteInstanceByProviderID(dropletID); err != nil {
		log.Error(err, "error deleting machine", "instance-id", dropletID)
	}

	do.eventRecorder.Eventf(machine, corev1.EventTypeNormal, "Deleted", "Deleted Machine %v", machine.Name)

	log.Info("Successfully deleted machine")
	return nil
}

func (do *MachineActuator) Update(_ context.Context, cluster *clusterv1.Cluster, goalMachine *clusterv1.Machine) error {
	log := do.Logger.WithValues("machine-name", goalMachine.Name)
	log.Info("call for updating machine")

	var err error

	goalConfig, err := doCapi.MachineConfigFromProviderSpec(goalMachine.Spec.ProviderSpec)
	if err != nil {
		log.Error(err, "failed to decode provider spec")
		return err
	}

	if err := do.validateMachine(goalConfig); err != nil {
		log.Error(err, "error validating machineconfig for machine")
		return err
	}

	exists, err := do.Exists(context.Background(), cluster, goalMachine)
	if err != nil {
		log.Error(err, "failed to check existance of machine")
		return err
	}

	if !exists {
		log.Info("vm not found, creating vm for machine")
		return do.Create(context.Background(), cluster, goalMachine)
	}

	if err := do.updateMachineStatus(goalMachine); err != nil {
		log.Error(err, "failed to update machine status")
		return err
	}

	log.Info("Updated machine successfully")
	return nil
}

func (do *MachineActuator) Exists(ctx context.Context, cluster *clusterv1.Cluster, machine *clusterv1.Machine) (bool, error) {
	log := do.Logger.WithValues("machine-name", machine.Name)
	log.Info("Checking existence of machine")
	var err error

	i, err := do.cm.conn.instanceIfExists(machine)
	if err != nil {
		log.Error(err, "error checking machine existance")
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
