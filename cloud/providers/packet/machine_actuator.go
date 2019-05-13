package packet

import (
	"context"
	"reflect"
	"strings"
	"time"

	"github.com/appscode/go/log"
	packetconfig "github.com/pharmer/pharmer/apis/v1beta1/packet"
	"github.com/pharmer/pharmer/cloud"
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

func (packet *MachineActuator) Create(_ context.Context, cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	log.Infof("creating machine %s", machine.Name)

	machineConfig, err := packetconfig.MachineConfigFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		return errors.Wrapf(err, "error decoding provider config for macine %s", machine.Name)
	}

	if err := packet.validateMachine(machineConfig); err != nil {
		return errors.Wrapf(err, "failed to valide machine config for machien %s", machine.Name)
	}

	if packet.conn, err = PrepareCloud(packet.ctx, cluster.Name, packet.owner); err != nil {
		return errors.Wrapf(err, "failed to prepare cloud")
	}
	exists, err := packet.Exists(packet.ctx, cluster, machine)
	if err != nil {
		return errors.Wrapf(err, "failed to check existance of machine %s", machine.Name)
	}

	if exists {
		log.Infof("Skipped creating a machine that already exists.")
	} else {
		log.Infof("vm not found, creating vm for machine %q", machine.Name)

		token, err := packet.getKubeadmToken()
		if err != nil {
			return errors.Wrap(err, "failed to generate kubeadm token")
		}

		_, err = packet.conn.CreateInstance(machine, token, packet.owner)
		if err != nil {
			return errors.Wrap(err, "failed to create instance")
		}
	}

	// set machine annotation
	sm := cloud.NewStatusManager(packet.client, packet.scheme)
	err = sm.UpdateInstanceStatus(machine)
	if err != nil {
		return errors.Wrap(err, "failed to set machine annotation")
	}

	// update machine provider status
	err = packet.updateMachineStatus(machine)
	if err != nil {
		return errors.Wrap(err, "failed to update machine status")
	}

	log.Infof("successfully created machine %s", machine.Name)
	return nil
}

func (packet *MachineActuator) updateMachineStatus(machine *clusterv1.Machine) error {
	vm, err := packet.conn.instanceIfExists(machine)
	if err != nil {
		return errors.Wrapf(err, "failed to check existance of machine %s", machine.Name)
	}

	status, err := packetconfig.MachineStatusFromProviderStatus(machine.Status.ProviderStatus)
	if err != nil {
		return errors.Wrapf(err, "failed to decode provider status of machine %s", machine.Name)
	}

	status.InstanceID = vm.ID

	raw, err := packetconfig.EncodeMachineStatus(status)
	if err != nil {
		return errors.Wrapf(err, "failed to encode provider status for machine %q", machine.Name)
	}
	machine.Status.ProviderStatus = raw

	if err := packet.client.Status().Update(packet.ctx, machine); err != nil {
		return errors.Wrapf(err, "failed to update provider status for machine %s", machine.Name)
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

func machineProviderFromProviderSpec(providerSpec clusterv1.ProviderSpec) (*packetconfig.PacketMachineProviderSpec, error) {
	var config packetconfig.PacketMachineProviderSpec
	if err := yaml.Unmarshal(providerSpec.Value.Raw, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func (packet *MachineActuator) Delete(_ context.Context, cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	fmt.Println("call for deleting machine")
	clusterName := cluster.Name
	if _, found := machine.Labels[clusterv1.MachineClusterLabelName]; found {
		clusterName = machine.Labels[clusterv1.MachineClusterLabelName]
	}
	var err error
	if packet.conn, err = PrepareCloud(packet.ctx, clusterName, packet.owner); err != nil {
		return err
	}
	instance, err := packet.conn.instanceIfExists(machine)
	if err != nil {
		// SKIp error
	}
	if instance == nil {
		log.Infof("Skipped deleting a VM that is already deleted")
		return nil
	}
	serverId := fmt.Sprintf("packet://%v", instance.ID)

	if err = packet.conn.DeleteInstanceByProviderID(serverId); err != nil {
		log.Infof("errror on deleting %v", err)
	}

	packet.eventRecorder.Eventf(machine, corev1.EventTypeNormal, "Deleted", "Deleted Machine %v", machine.Name)

	return nil
}

func (packet *MachineActuator) Update(_ context.Context, cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	log.Infof("updating machine %s", machine.Name)

	var err error
	if packet.conn, err = PrepareCloud(packet.ctx, cluster.Name, packet.owner); err != nil {
		return errors.Wrapf(err, "failed to prepare clou")
	}

	goalConfig, err := machineProviderFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		return errors.Wrapf(err, "failed to decode machine provider spec")
	}

	if err := packet.validateMachine(goalConfig); err != nil {
		return errors.Wrap(err, "failed to validate machine")
	}

	exists, err := packet.Exists(packet.ctx, cluster, machine)
	if err != nil {
		return errors.Wrapf(err, "failed to check existance of machine %s", machine.Name)
	}

	if !exists {
		log.Infof("vm not found, creating vm for machine %q", machine.Name)
		return packet.Create(packet.ctx, cluster, machine)
	}

	sm := NewStatusManager(packet.client, packet.scheme)
	status, err := sm.InstanceStatus(machine)
	if err != nil {
		return err
	}

	currentMachine := (*clusterv1.Machine)(status)
	if currentMachine == nil {
		log.Infof("status annotation not set, setting annotation")
		return sm.UpdateInstanceStatus(machine)
	}

	if !packet.requiresUpdate(currentMachine, machine) {
		fmt.Println("Don't require update")
		return nil
	}

	pharmerCluster, err := Store(packet.ctx).Owner(packet.owner).Clusters().Get(cluster.Name)
	if err != nil {
		return errors.Wrap(err, "failed to get pharmercluster")
	}

	kc, err := GetKubernetesClient(packet.ctx, pharmerCluster, packet.owner)
	if err != nil {
		return errors.Wrap(err, "failed to get kubeclient")
	}
	upm := NewUpgradeManager(packet.ctx, kc, packet.conn.cluster, packet.owner)
	if util.IsControlPlaneMachine(currentMachine) {
		if currentMachine.Spec.Versions.ControlPlane != machine.Spec.Versions.ControlPlane {
			log.Infof("Doing an in-place upgrade for master.\n")
			if err := upm.MasterUpgrade(currentMachine, machine); err != nil {
				return errors.Wrap(err, "failed to upgrade master")
			}
		}
	} else {
		//TODO(): Do we replace node or inplace upgrade?
		log.Infof("Doing an in-place upgrade for master.\n")
		if err := upm.NodeUpgrade(currentMachine, machine); err != nil {
			return errors.Wrap(err, "failed to upgrade node")
		}
	}

	if err := packet.updateMachineStatus(machine); err != nil {
		return errors.Wrap(err, "failed to update machine status")
	}

	log.Infof("Successfully updated machine %q", machine.Name)
	return nil
}

func (packet *MachineActuator) Exists(ctx context.Context, cluster *clusterv1.Cluster, machine *clusterv1.Machine) (bool, error) {
	fmt.Println("call for checking machine existence", machine.Name)
	clusterName := cluster.Name
	if _, found := machine.Labels[clusterv1.MachineClusterLabelName]; found {
		clusterName = machine.Labels[clusterv1.MachineClusterLabelName]
	}
	var err error
	if packet.conn, err = PrepareCloud(packet.ctx, clusterName, packet.owner); err != nil {
		return false, err
	}
	i, err := packet.conn.instanceIfExists(machine)
	if err != nil {
		return false, nil
	}

	return i != nil, nil
}

func (packet *MachineActuator) updateAnnotations(machine *clusterv1.Machine) error {
	name := machine.ObjectMeta.Name
	zone := packet.conn.cluster.Spec.Config.Cloud.Zone

	if machine.ObjectMeta.Annotations == nil {
		machine.ObjectMeta.Annotations = make(map[string]string)
	}
	machine.ObjectMeta.Annotations["zone"] = zone
	machine.ObjectMeta.Annotations["name"] = name
	err := packet.client.Update(context.Background(), machine)
	if err != nil {
		return err
	}
	if packet.client != nil {
		sm := NewStatusManager(packet.client, packet.scheme)
		return sm.UpdateInstanceStatus(machine)
	}
	return nil
}

// The two machines differ in a way that requires an update
func (packet *MachineActuator) requiresUpdate(a *clusterv1.Machine, b *clusterv1.Machine) bool {
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
