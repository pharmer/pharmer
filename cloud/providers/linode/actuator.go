package linode

import (
	"fmt"
	"reflect"
	"strconv"

	api "github.com/pharmer/pharmer/apis/v1"
	. "github.com/pharmer/pharmer/cloud"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	kubeadmconsts "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	client "sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset/typed/cluster/v1alpha1"
)

type Actuator struct {
	machineClient client.MachineInterface
	scheme        *runtime.Scheme
	codecFactory  *serializer.CodecFactory
}

func (cm *ClusterManager) InitializeActuator(machineClient client.MachineInterface) error {
	scheme, codecFactory, err := api.NewSchemeAndCodecs()
	if err != nil {
		return err
	}
	cm.actuator = &Actuator{
		machineClient: machineClient,
		scheme:        scheme,
		codecFactory:  codecFactory,
	}

	return nil
}

func (cm *ClusterManager) Create(cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	Logger(cm.ctx).Infoln("call for creating machine")
	if err := cm.PrepareCloud(cluster.Name); err != nil {
		return err
	}
	exists, err := cm.Exists(machine)
	if err != nil {
		return err
	}
	if !exists {
		token := ""
		if !IsMaster(machine) {
			kc, err := cm.GetAdminClient()
			if err != nil {
				return err
			}
			token, err = GetExistingKubeadmToken(kc, kubeadmconsts.DefaultTokenDuration)
			if err != nil {
				return err
			}
		}
		if !dryRun {
			if _, err = cm.conn.createOrUpdateStackScript(masterNG, ""); err != nil {
				return
			}
		}
		instance, err := cm.conn.CreateInstance(cm.cluster, machine, token)
		if err != nil {
			return err
		}

		if IsMaster(machine) {
			var providerConf *api.MachineProviderConfig
			providerConf, err = cm.cluster.MachineProviderConfig(machine)
			if err != nil {
				return err
			}
			if instance.PublicIP != "" {
				cluster.Status.APIEndpoints = append(cluster.Status.APIEndpoints, clusterv1.APIEndpoint{
					Host: instance.PublicIP,
					Port: int(cm.cluster.Spec.API.BindPort),
				})
			}
			if _, found := machine.Labels[api.PharmerHASetup]; found {
				did, err := strconv.Atoi(instance.ExternalID)
				if err != nil {
					return err
				}
				if err = cm.conn.addNodeToBalancer(cm.ctx, cm.namer.LoadBalancerName(), did); err != nil {
					return err
				}
			}
			cm.cluster.Spec.ETCDServers = append(cm.cluster.Spec.ETCDServers, instance.PublicIP)
		}

		if cm.actuator.machineClient != nil {
			return cm.updateAnnotations(machine)
		}
	} else {
		Logger(cm.ctx).Infoln("Skipped creating a machine that already exists.")
	}
	return nil
}

func (cm *ClusterManager) Exists(machine *clusterv1.Machine) (bool, error) {
	Logger(cm.ctx).Infoln("call for checking machine existence")
	clusterName := machine.ClusterName
	if _, found := machine.Labels[api.PharmerCluster]; found {
		clusterName = machine.Labels[api.PharmerCluster]
	}
	if err := cm.PrepareCloud(clusterName); err != nil {
		return false, err
	}
	i, err := cm.conn.instanceIfExists(machine)
	if err != nil {
		return false, nil
	}
	return i != nil, nil
}

func (cm *ClusterManager) updateAnnotations(machine *clusterv1.Machine) error {
	//	config, err := cloud.GetProviderconfig(cm.codecFactory, machine.Spec.ProviderConfig)
	name := machine.ObjectMeta.Name
	zone := cm.cluster.ProviderConfig().Zone

	if machine.ObjectMeta.Annotations == nil {
		machine.ObjectMeta.Annotations = make(map[string]string)
	}
	//machine.ObjectMeta.Annotations[ProjectAnnotationKey] = project
	machine.ObjectMeta.Annotations["zone"] = zone
	machine.ObjectMeta.Annotations["name"] = name
	_, err := cm.actuator.machineClient.Update(machine)
	if err != nil {
		return err
	}
	return cm.updateInstanceStatus(machine)
}

// Sets the status of the instance identified by the given machine to the given machine
func (cm *ClusterManager) updateInstanceStatus(machine *clusterv1.Machine) error {
	fmt.Println("updating instance status")
	sm := NewStatusManager(cm.actuator.machineClient, cm.actuator.scheme)
	status := sm.Initialize(machine)
	currentMachine, err := GetCurrentMachineIfExists(cm.actuator.machineClient, machine)
	if err != nil {
		return err
	}

	if currentMachine == nil {
		// The current status no longer exists because the matching CRD has been deleted.
		return fmt.Errorf("Machine has already been deleted. Cannot update current instance status for machine %v", machine.ObjectMeta.Name)
	}

	m, err := sm.SetMachineInstanceStatus(currentMachine, status)
	if err != nil {
		return err
	}

	_, err = cm.actuator.machineClient.Update(m)
	return err
}

// The two machines differ in a way that requires an update
func (cm *ClusterManager) requiresUpdate(a *clusterv1.Machine, b *clusterv1.Machine) bool {
	// Do not want status changes. Do want changes that impact machine provisioning
	return !reflect.DeepEqual(a.Spec.ObjectMeta, b.Spec.ObjectMeta) ||
		!reflect.DeepEqual(a.Spec.ProviderConfig, b.Spec.ProviderConfig) ||
		!reflect.DeepEqual(a.Spec.Roles, b.Spec.Roles) ||
		!reflect.DeepEqual(a.Spec.Versions, b.Spec.Versions) ||
		a.ObjectMeta.Name != b.ObjectMeta.Name ||
		a.ObjectMeta.UID != b.ObjectMeta.UID
}
