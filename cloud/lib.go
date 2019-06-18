package cloud

import (
	"time"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

var managedProviders = sets.NewString("aks", "gke", "eks", "dokube")

func GetSSHConfig(nodeName string, cluster *api.Cluster) (*api.SSHConfig, error) {
	//var err error
	//ctx, err = LoadCACertificates(ctx, Cluster, owner)
	//if err != nil {
	//	return nil, err
	//}
	//client, err := NewAdminClient(ctx, Cluster)
	//if err != nil {
	//	return nil, err
	//}
	//node, err := client.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
	//if err != nil {
	//	return nil, err
	//}
	//ctx, err = LoadSSHKey(ctx, Cluster, owner)
	//if err != nil {
	//	return nil, err
	//}
	//
	//cm, err := GetCloudManager(Cluster.ClusterConfig().Cloud.CloudProvider)
	//if err != nil {
	//	return nil, err
	//}
	//return cm.GetSSHConfig(Cluster, node)
	return nil, nil
}

// TODO: move
func UpdateGeneration(clusterStore store.ClusterStore, cluster *api.Cluster) (*api.Cluster, error) {
	if cluster == nil {
		return nil, errors.New("missing Cluster")
	} else if cluster.Name == "" {
		return nil, errors.New("missing Cluster name")
	} else if cluster.ClusterConfig().KubernetesVersion == "" {
		return nil, errors.New("missing Cluster version")
	}

	existing, err := clusterStore.Get(cluster.Name)
	if err != nil {
		return nil, errors.Errorf("Cluster `%s` does not exist. Reason: %v", cluster.Name, err)
	}
	cluster.Status = existing.Status
	cluster.Generation = time.Now().UnixNano()

	return clusterStore.Update(cluster)
}

func GetLeaderMachine(machineStore store.MachineStore, clusterName string) (*clusterapi.Machine, error) {
	machine, err := machineStore.Get(clusterName + "-master-0")
	if err != nil {
		return nil, err
	}
	return machine, nil
}
