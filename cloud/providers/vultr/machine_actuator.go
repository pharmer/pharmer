package vultr

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/appscode/go/log"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/apis/v1beta1/vultr"
	vultrconfig "github.com/pharmer/pharmer/apis/v1beta1/vultr"
	"github.com/pharmer/pharmer/cloud"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/cloud/machinesetup"
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
	AddToManagerFuncs = append(AddToManagerFuncs, func(ctx context.Context, m manager.Manager, owner string) error {
		actuator := NewMachineActuator(MachineActuatorParams{
			Ctx:           ctx,
			EventRecorder: m.GetEventRecorderFor(Recorder),
			Client:        m.GetClient(),
			Scheme:        m.GetScheme(),
			Owner:         owner,
		})
		return machine.AddWithActuator(m, actuator)
	})
}

type VultrClientKubeadm interface {
	TokenCreate(params kubeadm.TokenCreateParams) (string, error)
}

type VultrClientMachineSetupConfigGetter interface {
	GetMachineSetupConfig() (machinesetup.MachineSetupConfig, error)
}

type MachineActuator struct {
	ctx                      context.Context
	conn                     *cloudConnector
	client                   client.Client
	kubeadm                  VultrClientKubeadm
	machineSetupConfigGetter VultrClientMachineSetupConfigGetter
	eventRecorder            record.EventRecorder
	scheme                   *runtime.Scheme

	owner string
}

type MachineActuatorParams struct {
	Ctx            context.Context
	Kubeadm        VultrClientKubeadm
	Client         client.Client
	CloudConnector *cloudConnector
	EventRecorder  record.EventRecorder
	Scheme         *runtime.Scheme
	Owner          string
}

func NewMachineActuator(params MachineActuatorParams) *MachineActuator {
	return &MachineActuator{
		ctx:           params.Ctx,
		conn:          params.CloudConnector,
		client:        params.Client,
		kubeadm:       getKubeadm(params),
		eventRecorder: params.EventRecorder,
		scheme:        params.Scheme,
		owner:         params.Owner,
	}
}

func (do *MachineActuator) Create(ctx context.Context, cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	log.Infof("Creating machine %q for cluster %q", machine.Name, cluster.Name)

	machineConfig, err := vultr.MachineConfigFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		return errors.Wrap(err, "failed to decode machine provider spec")
	}

	if err := do.validateMachine(machineConfig); err != nil {
		log.Debugf("Error validating machineconfig for machine %q: %v", machine.Name, err)
		return err
	}

	if do.conn, err = PrepareCloud(do.ctx, cluster.Name, do.owner); err != nil {
		log.Debugf("Error in PrepareCloud for machine %q: %v", machine.Name, err)
		return err
	}
	exists, err := do.Exists(do.ctx, cluster, machine)
	if err != nil {
		return errors.Wrapf(err, "failed to check machine existance of %q", machine.Name)
	}

	if exists {
		log.Infof("Machine %q exists, skipping machine creation", machine.Name)
	} else {
		log.Infof("Droplet not found, creating droplet for machine %q", machine.Name)

		token, err := do.getKubeadmToken()
		if err != nil {
			log.Debugf("Error creating kubeadm token for machine %q: %v", machine.Name, err)
			return err
		}

		if _, err := do.conn.CreateInstance(machine.Name, token, machine, do.owner); err != nil {
			log.Debugf("Error creating instance for machine %q: %v", machine.Name, err)
			return err
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
		return errors.Wrap(err, "failed to update machine status")
	}

	log.Infof("Successfully created machine %q", machine.Name)
	return nil
}

func (do *MachineActuator) updateMachineStatus(machine *clusterv1.Machine) error {
	droplet, err := do.conn.instanceIfExists(machine)
	if err != nil {
		log.Debugf("Error checking existance for machine %q: %v", machine.Name, err)
		return err
	}

	status, err := vultr.MachineStatusFromProviderStatus(machine.Status.ProviderStatus)
	if err != nil {
		log.Debugf("Error getting machine status for %q: %v", machine.Name, err)
		return err
	}

	status.InstanceID = droplet.ID
	status.InstanceStatus = droplet.Status

	raw, err := vultr.EncodeMachineStatus(status)
	if err != nil {
		log.Debugf("Error encoding machine status for machine %q", machine.Name)
	}

	machine.Status.ProviderStatus = raw

	if err := do.client.Status().Update(do.ctx, machine); err != nil {
		return errors.Wrap(err, "failed to update machine status")
	}

	return nil
}

func (do *MachineActuator) validateMachine(providerConfig *vultrconfig.VultrMachineProviderSpec) error {
	if len(providerConfig.Image) == 0 {
		return errors.New("image slug must be provided")
	}
	if len(providerConfig.Region) == 0 {
		return errors.New("region must be provided")
	}
	if len(providerConfig.Plan) == 0 {
		return errors.New("type must be provided")
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
	log.Infof("call for deleting machine")
	clusterName := cluster.Name
	if _, found := machine.Labels[clusterv1.MachineClusterLabelName]; found {
		clusterName = machine.Labels[clusterv1.MachineClusterLabelName]
	}
	var err error
	if do.conn, err = PrepareCloud(do.ctx, clusterName, do.owner); err != nil {
		return err
	}
	instance, err := do.conn.instanceIfExists(machine)
	if err != nil {
		// SKIp error
	}
	if instance == nil {
		log.Infof("Skipped deleting a VM that is already deleted.\n")
		return nil
	}
	instanceID := fmt.Sprintf("vultr://%v", instance.ID)

	err = do.conn.deleteStartupScript(instance.Name, string(api.NodeRole))
	if err != nil {
		Logger(do.ctx).Infof("Failed to delete startup script. Reason: %s", err)
	}

	if err = do.conn.DeleteInstanceByProviderID(instanceID); err != nil {
		log.Infof("errror on deleting %v", err)
	}

	do.eventRecorder.Eventf(machine, corev1.EventTypeNormal, "Deleted", "Deleted Machine %v", machine.Name)

	return nil
}

func (do *MachineActuator) Update(_ context.Context, cluster *clusterv1.Cluster, goalMachine *clusterv1.Machine) error {
	log.Infof("Updating machine %q in cluster %q", goalMachine.Name, cluster.Name)

	var err error
	if do.conn, err = PrepareCloud(do.ctx, cluster.Name, do.owner); err != nil {
		return errors.Wrap(err, "failed to prepare cloud")
	}

	goalConfig, err := vultr.MachineConfigFromProviderSpec(goalMachine.Spec.ProviderSpec)
	if err != nil {
		return errors.Wrap(err, "failed to decode provider spec")
	}

	if err := do.validateMachine(goalConfig); err != nil {
		log.Debugf("Error validating machineconfig for machine %q: %v", goalMachine.Name, err)
		return err
	}

	exists, err := do.Exists(do.ctx, cluster, goalMachine)
	if err != nil {
		return errors.Wrapf(err, "failed to check existance of machine %s", goalMachine.Name)
	}

	if !exists {
		log.Infof("vm not found, creating vm for machine %q", goalMachine.Name)
		return do.Create(do.ctx, cluster, goalMachine)
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

	pharmerCluster, err := Store(do.ctx).Owner(do.owner).Clusters().Get(cluster.Name)
	if err != nil {
		return errors.Wrap(err, "failed to get pharmercluster")
	}

	kc, err := GetKubernetesClient(do.ctx, pharmerCluster, do.owner)
	if err != nil {
		return errors.Wrap(err, "failed to get kubeclient")
	}
	upm := NewUpgradeManager(do.ctx, kc, do.conn.cluster, do.owner)
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
	log.Infof("call for checking machine existence", machine.Name)
	clusterName := cluster.Name
	if _, found := machine.Labels[clusterv1.MachineClusterLabelName]; found {
		clusterName = machine.Labels[clusterv1.MachineClusterLabelName]
	}
	var err error
	if do.conn, err = PrepareCloud(do.ctx, clusterName, do.owner); err != nil {
		return false, err
	}
	i, err := do.conn.instanceIfExists(machine)
	if err != nil {
		return false, nil
	}

	host, err := do.conn.client.GetServer(i.ID)
	if err != nil {
		return false, err
	}

	if util.IsControlPlaneMachine(machine) {
		publicIP := host.MainIP
		if publicIP != "" {
			cluster.Status.APIEndpoints = []clusterv1.APIEndpoint{
				{
					Host: publicIP,
					Port: 6443,
				},
			}
		}
		if err = vultrconfig.SetVultrClusterProviderStatus(cluster); err != nil {
			return false, err
		}
		if err = do.client.Status().Update(ctx, cluster); err != nil {
			return false, err
		}
	}

	return i != nil, nil
}

func (do *MachineActuator) updateAnnotations(machine *clusterv1.Machine) error {
	name := machine.ObjectMeta.Name
	zone := do.conn.cluster.Spec.Config.Cloud.Zone

	if machine.ObjectMeta.Annotations == nil {
		machine.ObjectMeta.Annotations = make(map[string]string)
	}
	machine.ObjectMeta.Annotations["zone"] = zone
	machine.ObjectMeta.Annotations["name"] = name
	err := do.client.Update(context.Background(), machine)
	if err != nil {
		return err
	}
	if do.client != nil {
		sm := NewStatusManager(do.client, do.scheme)
		return sm.UpdateInstanceStatus(machine)
	}
	return nil
}

// The two machines differ in a way that requires an update
func (do *MachineActuator) requiresUpdate(a *clusterv1.Machine, b *clusterv1.Machine) bool {
	// Do not want status changes. Do want changes that impact machine provisioning
	return !reflect.DeepEqual(a.Spec.ObjectMeta, b.Spec.ObjectMeta) ||
		!reflect.DeepEqual(a.Spec.ProviderSpec, b.Spec.ProviderSpec) ||
		!reflect.DeepEqual(a.Spec.Versions, b.Spec.Versions) ||
		a.ObjectMeta.Name != b.ObjectMeta.Name
}

func getKubeadm(params MachineActuatorParams) VultrClientKubeadm {
	if params.Kubeadm == nil {
		return kubeadm.New()
	}
	return params.Kubeadm
}
