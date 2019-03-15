package packet

import (
	"context"
	"reflect"
	"strings"
	"time"

	packetconfig "github.com/pharmer/pharmer/apis/v1beta1/packet"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/cloud/machinesetup"
	"github.com/pkg/errors"
	"k8s.io/client-go/tools/record"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/controller/machine"
	"sigs.k8s.io/cluster-api/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	//	"k8s.io/client-go/kubernetes"
	"fmt"

	"github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	//kubeadmconsts "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"sigs.k8s.io/cluster-api/pkg/kubeadm"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

type DOClientKubeadm interface {
	TokenCreate(params kubeadm.TokenCreateParams) (string, error)
}

type DOClientMachineSetupConfigGetter interface {
	GetMachineSetupConfig() (machinesetup.MachineSetupConfig, error)
}

type MachineActuator struct {
	ctx                      context.Context
	conn                     *cloudConnector
	client                   client.Client
	kubeadm                  DOClientKubeadm
	machineSetupConfigGetter DOClientMachineSetupConfigGetter
	eventRecorder            record.EventRecorder
	scheme                   *runtime.Scheme

	owner string
}

type MachineActuatorParams struct {
	Ctx            context.Context
	Kubeadm        DOClientKubeadm
	Client         client.Client
	CloudConnector *cloudConnector
	EventRecorder  record.EventRecorder
	Scheme         *runtime.Scheme

	Owner string
}

func NewMachineActuator(params MachineActuatorParams) *MachineActuator {
	return &MachineActuator{
		ctx:           params.Ctx,
		conn:          params.CloudConnector,
		client:        params.Client,
		kubeadm:       getKubeadm(params),
		eventRecorder: params.EventRecorder,
		scheme:        params.Scheme,

		owner: params.Owner,
	}
}

func (do *MachineActuator) Create(_ context.Context, cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	Logger(do.ctx).Infoln("call for creating machine", machine.Name)
	machineConfig, err := machineProviderFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		return fmt.Errorf("error decoding provided machineConfig: %v", err)
	}

	if verr := do.validateMachine(machineConfig); err != nil {
		return verr
	}

	if do.conn, err = PrepareCloud(do.ctx, cluster.Name, do.owner); err != nil {
		return err
	}
	exists, err := do.Exists(do.ctx, cluster, machine)
	if err != nil {
		return err
	}

	if exists {
		fmt.Println("Skipped creating a machine that already exists.")
		return nil
	}

	token, err := do.getKubeadmToken()
	if err != nil {
		return err
	}

	instance, err := do.conn.CreateInstance(machine.Name, token, machine, do.owner)
	if err != nil {
		return err
	}

	if util.IsControlPlaneMachine(machine) {
		if instance.PublicIP != "" {
			cluster.Status.APIEndpoints = append(cluster.Status.APIEndpoints, clusterv1.APIEndpoint{
				Host: instance.PublicIP,
				Port: 6443,
			})
		}
	}

	if do.client != nil {
		sm := NewStatusManager(do.client, do.scheme)
		return sm.UpdateInstanceStatus(machine)
	}
	return nil
}

func (do *MachineActuator) validateMachine(providerConfig *packetconfig.PacketMachineProviderConfig) error {
	if len(providerConfig.Plan) == 0 {
		return errors.New("plan must be provided")
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

func machineProviderFromProviderSpec(providerSpec clusterv1.ProviderSpec) (*packetconfig.PacketMachineProviderConfig, error) {
	var config packetconfig.PacketMachineProviderConfig
	if err := yaml.Unmarshal(providerSpec.Value.Raw, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func (do *MachineActuator) Delete(_ context.Context, cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	fmt.Println("call for deleting machine")
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
		fmt.Println("Skipped deleting a VM that is already deleted.\n")
		return nil
	}
	serverId := fmt.Sprintf("packet://%v", instance.ID)

	if err = do.conn.DeleteInstanceByProviderID(serverId); err != nil {
		fmt.Println("errror on deleting %v", err)
	}

	do.eventRecorder.Eventf(machine, corev1.EventTypeNormal, "Deleted", "Deleted Machine %v", machine.Name)

	return nil
}

func (do *MachineActuator) Update(_ context.Context, cluster *clusterv1.Cluster, goalMachine *clusterv1.Machine) error {
	fmt.Println("call for updating machine")

	var err error
	if do.conn, err = PrepareCloud(do.ctx, cluster.Name, do.owner); err != nil {
		return err
	}

	goalConfig, err := machineProviderFromProviderSpec(goalMachine.Spec.ProviderSpec)
	if err != nil {
		return err
	}

	if verr := do.validateMachine(goalConfig); verr != nil {
		return err
	}

	sm := NewStatusManager(do.client, do.scheme)
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

	pharmerCluster, err := Store(do.ctx).Owner(do.owner).Clusters().Get(cluster.Name)
	if err != nil {
		return err
	}

	if !do.requiresUpdate(currentMachine, goalMachine) {
		fmt.Println("Don't require update")
		return nil
	}

	kc, err := GetKubernetesClient(do.ctx, pharmerCluster, do.owner)
	if err != nil {
		return err
	}

	upm := NewUpgradeManager(do.ctx, kc, do.conn.cluster, do.owner)
	if util.IsControlPlaneMachine(currentMachine) {
		if currentMachine.Spec.Versions.ControlPlane != goalMachine.Spec.Versions.ControlPlane {
			fmt.Println("Doing an in-place upgrade for master.\n")
			if err := upm.MasterUpgrade(currentMachine, goalMachine); err != nil {
				return err
			}
		}
	} else {
		//TODO(): Do we replace node or inplace upgrade?
		Logger(do.ctx).Infof("Doing an in-place upgrade for master.\n")
		if err := upm.NodeUpgrade(currentMachine, goalMachine); err != nil {
			return err
		}
	}
	return sm.UpdateInstanceStatus(goalMachine)
}

func (do *MachineActuator) Exists(ctx context.Context, cluster *clusterv1.Cluster, machine *clusterv1.Machine) (bool, error) {
	fmt.Println("call for checking machine existence", machine.Name)
	clusterName := cluster.Name
	if _, found := machine.Labels[clusterv1.MachineClusterLabelName]; found {
		clusterName = machine.Labels[clusterv1.MachineClusterLabelName]
	}
	var err error
	if do.conn, err = PrepareCloud(do.ctx, clusterName, do.owner); err != nil {
		return false, err
	}
	i, err := do.conn.instanceIfExists(machine)
	fmt.Println(err)
	if err != nil {
		return false, nil
	}

	if util.IsControlPlaneMachine(machine) {
		publicIP := i.Network[0].Address
		if publicIP != "" {
			cluster.Status.APIEndpoints = []clusterv1.APIEndpoint{
				{
					Host: publicIP,
					Port: 6443,
				},
			}
		}
		if err = packetconfig.SetPacketClusterProviderStatus(cluster); err != nil {
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

func getKubeadm(params MachineActuatorParams) DOClientKubeadm {
	if params.Kubeadm == nil {
		return kubeadm.New()
	}
	return params.Kubeadm
}
