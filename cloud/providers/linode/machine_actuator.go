package linode

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/appscode/go/log"
	linodeconfig "github.com/pharmer/pharmer/apis/v1beta1/linode"
	"github.com/pharmer/pharmer/cloud"
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

type LinodeClientKubeadm interface {
	TokenCreate(params kubeadm.TokenCreateParams) (string, error)
}

type LinodeClientMachineSetupConfigGetter interface {
	GetMachineSetupConfig() (machinesetup.MachineSetupConfig, error)
}

type MachineActuator struct {
	ctx                      context.Context
	conn                     *cloudConnector
	client                   client.Client
	kubeadm                  LinodeClientKubeadm
	machineSetupConfigGetter LinodeClientMachineSetupConfigGetter
	eventRecorder            record.EventRecorder
	scheme                   *runtime.Scheme

	owner string
}

type MachineActuatorParams struct {
	Ctx            context.Context
	Kubeadm        LinodeClientKubeadm
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
		//machineSetupConfigGetter: MachineSetup(params.Ctx),
	}
}

func (li *MachineActuator) Create(ctx context.Context, cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	log.Infof("creating machine %s", machine.Name)

	machineConfig, err := linodeconfig.MachineConfigFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		return errors.Wrapf(err, "error decoding provider config for macine %s", machine.Name)
	}

	if err := li.validateMachine(machineConfig); err != nil {
		return errors.Wrapf(err, "failed to valide machine config for machien %s", machine.Name)
	}

	if li.conn, err = PrepareCloud(li.ctx, cluster.Name, li.owner); err != nil {
		return errors.Wrapf(err, "failed to prepare cloud")
	}
	exists, err := li.Exists(li.ctx, cluster, machine)
	if err != nil {
		return errors.Wrapf(err, "failed to check existance of machine %s", machine.Name)
	}

	if exists {
		log.Infof("Skipped creating a machine that already exists.")
	} else {
		log.Infof("vm not found, creating vm for machine %q", machine.Name)

		token, err := li.getKubeadmToken()
		if err != nil {
			return errors.Wrap(err, "failed to generate kubeadm token")
		}

		server, err := li.conn.CreateInstance(machine, token)
		if err != nil {
			return errors.Wrap(err, "failed to create instance")
		}

		if util.IsControlPlaneMachine(machine) {
			if err = li.conn.addNodeToBalancer(li.conn.namer.LoadBalancerName(), machine.Name, server.PrivateIP); err != nil {
				return errors.Wrap(err, "failed to add machine to load balancer")
			}
		}
	}

	// set machine annotation
	sm := cloud.NewStatusManager(li.client, li.scheme)
	err = sm.UpdateInstanceStatus(machine)
	if err != nil {
		return errors.Wrap(err, "failed to set machine annotation")
	}

	// update machine provider status
	err = li.updateMachineStatus(machine)
	if err != nil {
		return errors.Wrap(err, "failed to update machine status")
	}

	log.Infof("successfully created machine %s", machine.Name)
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
	log.Infof("deleting machine %s", machine.Name)

	clusterName := cluster.Name
	if _, found := machine.Labels[clusterv1.MachineClusterLabelName]; found {
		clusterName = machine.Labels[clusterv1.MachineClusterLabelName]
	}
	var err error
	if li.conn, err = PrepareCloud(li.ctx, clusterName, li.owner); err != nil {
		return errors.Wrapf(err, "failed to prepare cloud")
	}
	instance, err := li.conn.instanceIfExists(machine)
	if err != nil {
		log.Infof("skipping error: %v", err)
	}
	if instance == nil {
		log.Infof("Skipped deleting a VM that is already deleted.\n")
		return nil
	}
	instanceID := fmt.Sprintf("linode://%v", instance.ID)

	if err = li.conn.DeleteInstanceByProviderID(instanceID); err != nil {
		log.Infof("errror on deleting %v", err)
	}

	li.eventRecorder.Eventf(machine, corev1.EventTypeNormal, "Deleted", "Deleted Machine %v", machine.Name)

	log.Infof("successfully deleted machine %s", machine.Name)
	return nil
}

func (li *MachineActuator) Update(_ context.Context, cluster *clusterv1.Cluster, goalMachine *clusterv1.Machine) error {
	log.Infof("updating machine %s", goalMachine.Name)

	var err error
	li.conn, err = PrepareCloud(li.ctx, cluster.Name, li.owner)
	if err != nil {
		return errors.Wrap(err, "failed to prepare cloud")
	}

	goalConfig, err := linodeconfig.MachineConfigFromProviderSpec(goalMachine.Spec.ProviderSpec)
	if err != nil {
		return errors.Wrap(err, "failed to decode provider spec")
	}

	if err := li.validateMachine(goalConfig); err != nil {
		return errors.Wrap(err, "failed to validate machine")
	}

	exists, err := li.Exists(li.ctx, cluster, goalMachine)
	if err != nil {
		return errors.Wrapf(err, "failed to check existance of machine %s", goalMachine.Name)
	}

	if !exists {
		log.Infof("vm not found, creating vm for machine %q", goalMachine.Name)
		return li.Create(li.ctx, cluster, goalMachine)
	}

	sm := cloud.NewStatusManager(li.client, li.scheme)
	status, err := sm.InstanceStatus(goalMachine)
	if err != nil {
		return err
	}

	currentMachine := (*clusterv1.Machine)(status)
	if currentMachine == nil {
		log.Infof("status annotation not set, setting annotation")
		return sm.UpdateInstanceStatus(goalMachine)
	}

	if !li.requiresUpdate(currentMachine, goalMachine) {
		log.Infof("Don't require update")
		return nil
	}

	pharmerCluster, err := cloud.Store(li.ctx).Owner(li.owner).Clusters().Get(cluster.Name)
	if err != nil {
		return errors.Wrap(err, "failed to get pharmercluster")
	}

	kc, err := cloud.GetKubernetesClient(li.ctx, pharmerCluster, li.owner)
	if err != nil {
		return errors.Wrap(err, "failed to get kubeclient")
	}
	upm := cloud.NewUpgradeManager(li.ctx, kc, li.conn.cluster, li.owner)
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

	if err := li.updateMachineStatus(goalMachine); err != nil {
		return errors.Wrap(err, "failed to update machine status")
	}

	log.Infof("Successfully updated machine %q", goalMachine.Name)
	return nil
}

func (li *MachineActuator) Exists(ctx context.Context, cluster *clusterv1.Cluster, machine *clusterv1.Machine) (bool, error) {
	log.Infof("checking existance of machine %s", machine.Name)

	clusterName := cluster.Name
	if _, found := machine.Labels[clusterv1.MachineClusterLabelName]; found {
		clusterName = machine.Labels[clusterv1.MachineClusterLabelName]
	}
	var err error
	if li.conn, err = PrepareCloud(li.ctx, clusterName, li.owner); err != nil {
		return false, errors.Wrapf(err, "failed to prepare pharmer cloud")
	}
	i, err := li.conn.instanceIfExists(machine)
	if err != nil {
		log.Infof("error checking machine existance: %v", err)
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
	vm, err := li.conn.instanceIfExists(machine)
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

	if err := li.client.Status().Update(li.ctx, machine); err != nil {
		return errors.Wrapf(err, "failed to update provider status for machine %s", machine.Name)
	}

	return nil
}

// The two machines differ in a way that requires an update
func (li *MachineActuator) requiresUpdate(a *clusterv1.Machine, b *clusterv1.Machine) bool {
	// Do not want status changes. Do want changes that impact machine provisioning
	return !reflect.DeepEqual(a.Spec.ObjectMeta, b.Spec.ObjectMeta) ||
		!reflect.DeepEqual(a.Spec.ProviderSpec, b.Spec.ProviderSpec) ||
		!reflect.DeepEqual(a.Spec.Versions, b.Spec.Versions) ||
		a.ObjectMeta.Name != b.ObjectMeta.Name
}
