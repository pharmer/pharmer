package digitalocean

import (
	. "github.com/pharmer/pharmer/cloud"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	//	"k8s.io/client-go/kubernetes"
	"fmt"
	"reflect"
	"strconv"

	api "github.com/pharmer/pharmer/apis/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	kubeadmconsts "k8s.io/kubernetes/cmd/kubeadm/app/constants"
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
			if providerConf.Config.ExternalIPType == api.IPTypeReserved {
				var reservedIP string
				reservedIP, err = cm.conn.createReserveIP()
				if err != nil {
					return err
				}
				id, _ := strconv.Atoi(instance.ExternalID)
				if err = cm.conn.assignReservedIP(reservedIP, id); err != nil {
					return err
				}
				cm.cluster.Status.ReservedIPs = append(cm.cluster.Status.ReservedIPs, api.ReservedIP{
					IP: reservedIP,
				})
				cluster.Status.APIEndpoints = append(cluster.Status.APIEndpoints, clusterv1.APIEndpoint{
					Host: reservedIP,
					Port: int(cm.cluster.Spec.API.BindPort),
				})
			} else if instance.PublicIP != "" {
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
		} else {
			cm.cluster.Spec.ClusterAPI = cluster
			Store(cm.ctx).Clusters().Update(cm.cluster)
		}
	} else {
		Logger(cm.ctx).Infoln("Skipped creating a machine that already exists.")
	}
	return nil
}

func (cm *ClusterManager) Delete(machine *clusterv1.Machine) error {
	Logger(cm.ctx).Infoln("call for deleting machine")
	clusterName := machine.ClusterName
	if _, found := machine.Labels[api.PharmerCluster]; found {
		clusterName = machine.Labels[api.PharmerCluster]
	}
	if err := cm.PrepareCloud(clusterName); err != nil {
		return err
	}
	instance, err := cm.conn.instanceIfExists(machine)
	if err != nil {
		return err
	}

	if instance == nil {
		Logger(cm.ctx).Infof("Skipped deleting a VM that is already deleted.\n")
		return nil
	}
	kc, err := cm.GetAdminClient()
	if err != nil {
		return err
	}

	node, err := kc.CoreV1().Nodes().Get(machine.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if err = cm.conn.DeleteInstanceByProviderID(node.Spec.ProviderID); err != nil {
		Logger(cm.ctx).Infoln("errror on deleting %v", err)
	}

	if cm.actuator.machineClient != nil {
		// Remove the finalizer
		machine.ObjectMeta.Finalizers = Filter(machine.ObjectMeta.Finalizers, clusterv1.MachineFinalizer)
		_, err = cm.actuator.machineClient.Update(machine)
		return err
	}

	return nil
}

func (cm *ClusterManager) Update(cluster *clusterv1.Cluster, goalMachine *clusterv1.Machine) error {
	Logger(cm.ctx).Infoln("call for updating machine")
	if err := cm.PrepareCloud(cluster.Name); err != nil {
		return err
	}

	sm := NewStatusManager(cm.actuator.machineClient, cm.actuator.scheme)
	status, err := sm.InstanceStatus(goalMachine)
	if err != nil {
		return err
	}

	currentMachine := (*clusterv1.Machine)(status)
	if currentMachine == nil {
		instance, err := cm.conn.instanceIfExists(goalMachine)
		if err != nil {
			return err
		}
		if instance != nil {
			Logger(cm.ctx).Infof("Populating current state for boostrap machine %v", goalMachine.ObjectMeta.Name)
			return cm.updateAnnotations(goalMachine)
		} else {
			return fmt.Errorf("cannot retrieve current state to update machine %v", goalMachine.ObjectMeta.Name)
		}
	}
	if !cm.requiresUpdate(currentMachine, goalMachine) {
		return nil
	}
	kc, err := cm.GetAdminClient()
	if err != nil {
		return err
	}

	upm := NewUpgradeManager(cm.ctx, cm, kc, cm.cluster)
	if IsMaster(currentMachine) {
		Logger(cm.ctx).Infof("Doing an in-place upgrade for master.\n")
		if err := upm.MasterUpgrade(currentMachine, goalMachine); err != nil {
			return err
		}
	} else {
		//TODO(): Do we replace node or inplace upgrade?
		Logger(cm.ctx).Infof("Doing an in-place upgrade for master.\n")
		if err := upm.NodeUpgrade(currentMachine, goalMachine); err != nil {
			return err
		}
	}
	return cm.updateInstanceStatus(goalMachine)
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
