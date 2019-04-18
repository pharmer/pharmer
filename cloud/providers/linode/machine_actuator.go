package linode

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	linodeconfig "github.com/pharmer/pharmer/apis/v1beta1/linode"
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
			EventRecorder: m.GetRecorder(Recorder),
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
	Logger(li.ctx).Infoln("call for creating machine", machine.Name)
	machineConfig, err := machineProviderFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		return fmt.Errorf("error decoding provided machineConfig: %v", err)
	}

	if verr := li.validateMachine(machineConfig); err != nil {
		return verr
	}

	if li.conn, err = PrepareCloud(li.ctx, cluster.Name, li.owner); err != nil {
		return err
	}
	exists, err := li.Exists(li.ctx, cluster, machine)
	if err != nil {
		return err
	}

	if exists {
		fmt.Println("Skipped creating a machine that already exists.")
		return nil
	}

	token, err := li.getKubeadmToken()
	if err != nil {
		return err
	}

	server, err := li.conn.CreateInstance(machine.Name, token, machine, li.owner)
	if err != nil {
		return err
	}

	if util.IsControlPlaneMachine(machine) {
		if err = li.conn.addNodeToBalancer(li.conn.namer.LoadBalancerName(), machine.Name, server.PrivateIP); err != nil {
			return err
		}
	}

	if li.client != nil {
		sm := NewStatusManager(li.client, li.scheme)
		return sm.UpdateInstanceStatus(machine)
	}
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

func machineProviderFromProviderSpec(providerSpec clusterv1.ProviderSpec) (*linodeconfig.LinodeMachineProviderSpec, error) {
	var config linodeconfig.LinodeMachineProviderSpec
	if err := yaml.Unmarshal(providerSpec.Value.Raw, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func (li *MachineActuator) Delete(_ context.Context, cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	fmt.Println("call for deleting machine")
	clusterName := cluster.Name
	if _, found := machine.Labels[clusterv1.MachineClusterLabelName]; found {
		clusterName = machine.Labels[clusterv1.MachineClusterLabelName]
	}
	var err error
	if li.conn, err = PrepareCloud(li.ctx, clusterName, li.owner); err != nil {
		return err
	}
	instance, err := li.conn.instanceIfExists(machine)
	if err != nil {
		// SKIp error
	}
	if instance == nil {
		fmt.Println("Skipped deleting a VM that is already deleted.\n")
		return nil
	}
	instanceID := fmt.Sprintf("linode://%v", instance.ID)

	if err = li.conn.DeleteInstanceByProviderID(instanceID); err != nil {
		fmt.Println("errror on deleting %v", err)
	}

	li.eventRecorder.Eventf(machine, corev1.EventTypeNormal, "Deleted", "Deleted Machine %v", machine.Name)

	return nil
}

func (li *MachineActuator) Update(_ context.Context, cluster *clusterv1.Cluster, goalMachine *clusterv1.Machine) error {
	fmt.Println("call for updating machine")

	var err error
	if li.conn, err = PrepareCloud(li.ctx, cluster.Name, li.owner); err != nil {
		return err
	}

	goalConfig, err := machineProviderFromProviderSpec(goalMachine.Spec.ProviderSpec)
	if err != nil {
		return err
	}

	if verr := li.validateMachine(goalConfig); verr != nil {
		return err
	}

	sm := NewStatusManager(li.client, li.scheme)
	status, err := sm.InstanceStatus(goalMachine)
	if err != nil {
		return err
	}
	if util.IsControlPlaneMachine(goalMachine) {
		if status == nil {
			return sm.UpdateInstanceStatus(goalMachine)
		}
	}

	currentMachine := (*clusterv1.Machine)(status)
	if currentMachine == nil {
		return errors.New("status annotation not set")
	}

	pharmerCluster, err := Store(li.ctx).Owner(li.owner).Clusters().Get(cluster.Name)
	if err != nil {
		return err
	}

	if !li.requiresUpdate(currentMachine, goalMachine) {
		fmt.Println("Don't require update")
		return nil
	}

	kc, err := GetKubernetesClient(li.ctx, pharmerCluster, li.owner)
	if err != nil {
		return err
	}

	upm := NewUpgradeManager(li.ctx, kc, li.conn.cluster, li.owner)
	if util.IsControlPlaneMachine(currentMachine) {
		if currentMachine.Spec.Versions.ControlPlane != goalMachine.Spec.Versions.ControlPlane {
			fmt.Println("Doing an in-place upgrade for master.\n")
			if err := upm.MasterUpgrade(currentMachine, goalMachine); err != nil {
				return err
			}
		}
	} else {
		//TODO(): Do we replace node or inplace upgrade?
		Logger(li.ctx).Infof("Doing an in-place upgrade for master.\n")
		if err := upm.NodeUpgrade(currentMachine, goalMachine); err != nil {
			return err
		}
	}
	return sm.UpdateInstanceStatus(goalMachine)
}

func (li *MachineActuator) Exists(ctx context.Context, cluster *clusterv1.Cluster, machine *clusterv1.Machine) (bool, error) {
	fmt.Println("call for checking machine existence", machine.Name)
	clusterName := cluster.Name
	if _, found := machine.Labels[clusterv1.MachineClusterLabelName]; found {
		clusterName = machine.Labels[clusterv1.MachineClusterLabelName]
	}
	var err error
	if li.conn, err = PrepareCloud(li.ctx, clusterName, li.owner); err != nil {
		return false, err
	}
	i, err := li.conn.instanceIfExists(machine)
	if err != nil {
		return false, nil
	}

	if util.IsControlPlaneMachine(machine) {
		publicIP := i.IPv4[0].String()
		if publicIP != "" {
			cluster.Status.APIEndpoints = []clusterv1.APIEndpoint{
				{
					Host: publicIP,
					Port: 6443,
				},
			}
		}
		if err = linodeconfig.SetLinodeClusterProviderStatus(cluster); err != nil {
			return false, err
		}
		if err = li.client.Status().Update(ctx, cluster); err != nil {
			return false, err
		}
	}

	return i != nil, nil
}

func (li *MachineActuator) updateAnnotations(machine *clusterv1.Machine) error {
	name := machine.ObjectMeta.Name
	zone := li.conn.cluster.Spec.Config.Cloud.Zone

	if machine.ObjectMeta.Annotations == nil {
		machine.ObjectMeta.Annotations = make(map[string]string)
	}
	machine.ObjectMeta.Annotations["zone"] = zone
	machine.ObjectMeta.Annotations["name"] = name
	err := li.client.Update(context.Background(), machine)
	if err != nil {
		return err
	}
	if li.client != nil {
		sm := NewStatusManager(li.client, li.scheme)
		return sm.UpdateInstanceStatus(machine)
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

func getKubeadm(params MachineActuatorParams) LinodeClientKubeadm {
	if params.Kubeadm == nil {
		return kubeadm.New()
	}
	return params.Kubeadm
}
