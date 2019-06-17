package digitalocean

import (
	"context"
	"reflect"
	"strings"
	"time"

	"github.com/appscode/go/log"
	"github.com/pharmer/pharmer/cloud"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	"k8s.io/client-go/tools/record"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/controller/machine"
	"sigs.k8s.io/cluster-api/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	//	"k8s.io/client-go/kubernetes"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

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
			Scheme:        m.GetScheme(),
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
	scheme        *runtime.Scheme
	cm            *ClusterManager
}

type MachineActuatorParams struct {
	Kubeadm        DOClientKubeadm
	Client         client.Client
	CloudConnector *cloudConnector
	EventRecorder  record.EventRecorder
	Scheme         *runtime.Scheme
	cm             *ClusterManager
}

func NewMachineActuator(params MachineActuatorParams) *MachineActuator {
	return &MachineActuator{
		client:        params.Client,
		kubeadm:       getKubeadm(params),
		eventRecorder: params.EventRecorder,
		scheme:        params.Scheme,
		cm:            params.cm,
	}
}

func (do *MachineActuator) Create(_ context.Context, cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	log.Infof("Creating machine %q for cluster %q", machine.Name, cluster.Name)

	machineConfig, err := doCapi.MachineConfigFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		return errors.Wrapf(err, "error decoding provider config for macine %s", machine.Name)
	}

	if err := do.validateMachine(machineConfig); err != nil {
		return errors.Wrapf(err, "failed to valide machine config for machien %s", machine.Name)
	}

	exists, err := do.Exists(context.Background(), cluster, machine)
	if err != nil {
		return errors.Wrapf(err, "failed to check existance of machine %s", machine.Name)
	}

	if exists {
		log.Infof("Machine %q exists, skipping machine creation", machine.Name)
	} else {
		log.Infof("Droplet not found, creating droplet for machine %q", machine.Name)

		token, err := do.getKubeadmToken()
		if err != nil {
			return errors.Wrap(err, "failed to generate kubeadm token")
		}

		script, err := RenderStartupScript(do.cm, machine, token, customTemplate)
		if err != nil {
			return err
		}

		if _, err := do.cm.conn.CreateInstance(do.cm.conn.Cluster, machine, script); err != nil {
			return errors.Wrap(err, "failed to create instance")
		}
	}

	// set machine annotation
	sm := cloud.NewStatusManager(do.client, do.scheme)
	err = sm.UpdateInstanceStatus(machine)
	if err != nil {
		return errors.Wrap(err, "failed to set machine annotation")
	}

	// update machine provider status
	err = do.updateMachineStatus(machine)
	if err != nil {
		return err
	}

	log.Infof("Successfully created machine %q", machine.Name)
	return nil
}

func (do *MachineActuator) updateMachineStatus(machine *clusterv1.Machine) error {
	droplet, err := do.cm.conn.instanceIfExists(machine)
	if err != nil {
		log.Debugf("Error checking existance for machine %q: %v", machine.Name, err)
		return err
	}

	status, err := doCapi.MachineStatusFromProviderStatus(machine.Status.ProviderStatus)
	if err != nil {
		log.Debugf("Error getting machine status for %q: %v", machine.Name, err)
		return err
	}

	status.InstanceID = droplet.ID
	status.InstanceStatus = droplet.Status

	raw, err := doCapi.EncodeMachineStatus(status)
	if err != nil {
		log.Debugf("Error encoding machine status for machine %q", machine.Name)
	}

	machine.Status.ProviderStatus = raw

	if err := do.client.Status().Update(context.Background(), machine); err != nil {
		log.Debugf("Error updating status for machine %q", machine.Name)
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
	log.Infof("Deleting machine %q in cluster %q", machine.Name, cluster.Name)
	var err error

	instance, err := do.cm.conn.instanceIfExists(machine)
	if err != nil {
		log.Infof("Error checking machine existance: %v", err)
	}
	if instance == nil {
		log.Infof("Skipped deleting a VM that is already deleted")
		return nil
	}
	dropletId := fmt.Sprintf("digitalocean://%v", instance.ID)

	if err = do.cm.conn.DeleteInstanceByProviderID(dropletId); err != nil {
		log.Infof("errror on deleting %v", err)
	}

	do.eventRecorder.Eventf(machine, corev1.EventTypeNormal, "Deleted", "Deleted Machine %v", machine.Name)

	log.Infof("Successfully deleted machine %q", machine.Name)
	return nil
}

func (do *MachineActuator) Update(_ context.Context, cluster *clusterv1.Cluster, goalMachine *clusterv1.Machine) error {
	log.Infof("Updating machine %q in cluster %q", goalMachine.Name, cluster.Name)

	var err error

	goalConfig, err := doCapi.MachineConfigFromProviderSpec(goalMachine.Spec.ProviderSpec)
	if err != nil {
		return errors.Wrap(err, "failed to decode provider spec")
	}

	if err := do.validateMachine(goalConfig); err != nil {
		log.Debugf("Error validating machineconfig for machine %q: %v", goalMachine.Name, err)
		return err
	}

	exists, err := do.Exists(context.Background(), cluster, goalMachine)
	if err != nil {
		return errors.Wrapf(err, "failed to check existance of machine %s", goalMachine.Name)
	}

	if !exists {
		log.Infof("vm not found, creating vm for machine %q", goalMachine.Name)
		return do.Create(context.Background(), cluster, goalMachine)
	}

	sm := NewStatusManager(do.client, do.scheme)
	status, err := sm.InstanceStatus(goalMachine)
	if err != nil {
		return errors.Wrapf(err, "failed to get instance status of machine %s", goalMachine.Name)
	}

	currentMachine := (*clusterv1.Machine)(status)
	if currentMachine == nil {
		log.Infof("status annotation not set, setting annotation")
		return sm.UpdateInstanceStatus(goalMachine)
	}

	if !do.requiresUpdate(currentMachine, goalMachine) {
		log.Infof("Don't require update")
		return nil
	}

	pharmerCluster, err := store.StoreProvider.Clusters().Get(cluster.Name)
	if err != nil {
		return errors.Wrap(err, "failed to get pharmercluster")
	}

	kc, err := GetKubernetesClient(do.cm, pharmerCluster)
	if err != nil {
		return errors.Wrap(err, "failed to get kubeclient")
	}
	upm := NewUpgradeManager(kc, do.cm.conn.Cluster)
	if util.IsControlPlaneMachine(currentMachine) {
		if currentMachine.Spec.Versions.ControlPlane != goalMachine.Spec.Versions.ControlPlane {
			log.Infof("Doing an in-place upgrade for master.\n")
			if err := upm.MasterUpgrade(currentMachine, goalMachine); err != nil {
				return errors.Wrap(err, "failed to upgrade master")
			}
		}
	} else {
		//TODO(): Do we replace node or inplace upgrade?
		log.Infof("Doing an in-place upgrade for master.\n")
		if err := upm.NodeUpgrade(currentMachine, goalMachine); err != nil {
			return errors.Wrap(err, "failed to upgrade node")
		}
	}

	if err := do.updateMachineStatus(goalMachine); err != nil {
		return errors.Wrap(err, "failed to update machine status")
	}

	log.Infof("Updated machine %q successfully", goalMachine.Name)
	return nil
}

func (do *MachineActuator) Exists(ctx context.Context, cluster *clusterv1.Cluster, machine *clusterv1.Machine) (bool, error) {
	log.Infof("Checking existance of machine %q in cluster %q", machine.Name, cluster.Name)
	var err error

	i, err := do.cm.conn.instanceIfExists(machine)
	if err != nil {
		log.Infof("Error checking machine existance: %v", err)
		return false, nil
	}

	return i != nil, nil
}

// The two machines differ in a way that requires an update
func (do *MachineActuator) requiresUpdate(a *clusterv1.Machine, b *clusterv1.Machine) bool {
	// Do not want status changes. Do want changes that impact machine provisioning
	return !reflect.DeepEqual(a.Spec.ObjectMeta, b.Spec.ObjectMeta) ||
		!reflect.DeepEqual(a.Spec.ProviderSpec, b.Spec.ProviderSpec) ||
		!reflect.DeepEqual(a.Spec.Versions, b.Spec.Versions) ||
		a.ObjectMeta.Name != b.ObjectMeta.Name
}

func getKubeadm(params MachineActuatorParams) DOClientKubeadm {
	if params.Kubeadm == nil {
		return kubeadm.New()
	}
	return params.Kubeadm
}
