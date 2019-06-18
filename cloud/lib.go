package cloud

import (
	"net"

	"github.com/appscode/go/term"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

var managedProviders = sets.NewString("aks", "gke", "eks", "dokube")

func GetSSHConfig(storeProvider store.ResourceInterface, clusterName, nodeName string) (*api.SSHConfig, error) {
	cluster, err := storeProvider.Clusters().Get(clusterName)
	if err != nil {
		return nil, err
	}

	scope := NewScope(NewScopeParams{
		Cluster:       cluster,
		StoreProvider: storeProvider,
	})
	cm, err := scope.GetCloudManager()
	if err != nil {
		return nil, err
	}

	client, err := cm.GetAdminClient()
	if err != nil {
		return nil, err
	}

	node, err := client.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
	term.ExitOnError(err)

	cfg := &api.SSHConfig{
		PrivateKey: scope.Certs.SSHKey.PrivateKey,
		User:       cluster.Spec.Config.SSHUserName,
		HostPort:   int32(22),
	}
	for _, addr := range node.Status.Addresses {
		if addr.Type == corev1.NodeExternalIP {
			cfg.HostIP = addr.Address
		}
	}
	if net.ParseIP(cfg.HostIP) == nil {
		return nil, errors.Errorf("failed to detect external Ip for node %s of cluster %s", node.Name, cluster.Name)
	}

	return cfg, nil
}

func getLeaderMachine(machineStore store.MachineStore, clusterName string) (*clusterapi.Machine, error) {
	machine, err := machineStore.Get(clusterName + "-master-0")
	if err != nil {
		return nil, err
	}
	return machine, nil
}
