package digitalocean

import (
	. "github.com/pharmer/pharmer/cloud"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	//	"k8s.io/client-go/kubernetes"
	"fmt"

	api "github.com/pharmer/pharmer/apis/v1"
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
	fmt.Println(machineClient)
	cm.actuator = &Actuator{
		machineClient: machineClient,
		scheme:        scheme,
		codecFactory:  codecFactory,
	}

	return nil
}

func (cm *ClusterManager) PrepareCloud(clusterName string) error {
	var err error

	cluster, err := Store(cm.ctx).Clusters().Get(clusterName)
	if err != nil {
		return fmt.Errorf("cluster `%s` does not exist. Reason: %v", clusterName, err)
	}
	cm.cluster = cluster

	if cm.ctx, err = LoadCACertificates(cm.ctx, cm.cluster); err != nil {
		return err
	}
	if cm.ctx, err = LoadSSHKey(cm.ctx, cm.cluster); err != nil {
		return err
	}
	if cm.conn, err = NewConnector(cm.ctx, cm.cluster); err != nil {
		return err
	}
	return nil
}

func (cm *ClusterManager) Create(cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	if err := cm.PrepareCloud(cluster.Name); err != nil {
		return err
	}
	exists, err := cm.Exists(machine)
	if err != nil {
		return err
	}
	if !exists {
		if IsMaster(machine) {
			Logger(cm.ctx).Infoln("master can be created by pharmer only")
			return nil
		}
		kc, err := cm.GetAdminClient()
		if err != nil {
			return err
		}
		token, err := GetExistingKubeadmToken(kc, kubeadmconsts.DefaultTokenDuration)
		if err != nil {
			return err
		}
		_, err = cm.conn.CreateInstance(cm.cluster, machine, token)
		if err != nil {
			return err
		}
	}

	Logger(cm.ctx).Infoln("Skipped creating a machine that already exists.")

	return nil
}

func (cm *ClusterManager) Delete(machine *clusterv1.Machine) error {
	if err := cm.PrepareCloud(machine.ClusterName); err != nil {
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

	err = cm.conn.deleteInstance(cm.ctx, instance.ID)
	if err != nil {
		return err
	}

	if cm.actuator.machineClient != nil {
		// Remove the finalizer
		machine.ObjectMeta.Finalizers = Filter(machine.ObjectMeta.Finalizers, clusterv1.MachineFinalizer)
		_, err = cm.actuator.machineClient.Update(machine)
	}

	return err
	return nil
}

func (cm *ClusterManager) Update(cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	if err := cm.PrepareCloud(cluster.Name); err != nil {
		return err
	}
	return nil
}

func (cm *ClusterManager) Exists(machine *clusterv1.Machine) (bool, error) {
	if err := cm.PrepareCloud(machine.ClusterName); err != nil {
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
	err = cm.updateInstanceStatus(machine)
	return err
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
