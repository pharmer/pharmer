package cloud

import (
	"fmt"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/clusterdeployer/clusterclient"

	"github.com/pharmer/pharmer/apis/v1beta1"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud/utils/certificates"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	"gomodules.xyz/cert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

var managedProviders = sets.NewString("aks", "gke", "eks", "dokube")

func List(opts metav1.ListOptions) ([]*api.Cluster, error) {
	return store.StoreProvider.Clusters().List(opts)
}

func GetCluster(name string) (*api.Cluster, error) {
	return store.StoreProvider.Clusters().Get(name)
}

func Delete(name string) (*api.Cluster, error) {
	if name == "" {
		return nil, errors.New("missing Cluster name")
	}

	cluster, err := store.StoreProvider.Clusters().Get(name)
	if err != nil {
		return nil, errors.Errorf("Cluster `%s` does not exist. Reason: %v", name, err)
	}
	cluster.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	cluster.Status.Phase = api.ClusterDeleting

	return store.StoreProvider.Clusters().Update(cluster)
}

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
func GetAdminConfig(cm Interface) (*api.KubeConfig, error) {
	cluster := cm.GetCluster()
	if managedProviders.Has(cluster.ClusterConfig().Cloud.CloudProvider) {
		return cm.GetKubeConfig()
	}
	var err error

	caCertPair := cm.GetCaCertPair()
	adminCert, adminKey, err := certificates.CreateAdminCertificate(caCertPair.Cert, caCertPair.Key)
	if err != nil {
		return nil, err
	}

	var (
		clusterName = fmt.Sprintf("%s.pharmer", cluster.Name)
		userName    = fmt.Sprintf("cluster-admin@%s.pharmer", cluster.Name)
		ctxName     = fmt.Sprintf("cluster-admin@%s.pharmer", cluster.Name)
	)
	cfg := api.KubeConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "KubeConfig",
		},
		Preferences: api.Preferences{
			Colors: true,
		},
		Cluster: api.NamedCluster{
			Name:                     clusterName,
			Server:                   cluster.APIServerURL(),
			CertificateAuthorityData: cert.EncodeCertPEM(caCertPair.Cert),
		},
		AuthInfo: api.NamedAuthInfo{
			Name:                  userName,
			ClientCertificateData: cert.EncodeCertPEM(adminCert),
			ClientKeyData:         cert.EncodePrivateKeyPEM(adminKey),
		},
		Context: api.NamedContext{
			Name:     ctxName,
			Cluster:  clusterName,
			AuthInfo: userName,
		},
	}
	return &cfg, nil
}

func UpdateSpec(cluster *api.Cluster) (*api.Cluster, error) {
	if cluster == nil {
		return nil, errors.New("missing Cluster")
	} else if cluster.Name == "" {
		return nil, errors.New("missing Cluster name")
	} else if cluster.ClusterConfig().KubernetesVersion == "" {
		return nil, errors.New("missing Cluster version")
	}

	existing, err := store.StoreProvider.Clusters().Get(cluster.Name)
	if err != nil {
		return nil, errors.Errorf("Cluster `%s` does not exist. Reason: %v", cluster.Name, err)
	}
	cluster.Status = existing.Status
	cluster.Generation = time.Now().UnixNano()

	return store.StoreProvider.Clusters().Update(cluster)
}

func GetBooststrapClient(cm Interface, cluster *api.Cluster) (clusterclient.Client, error) {
	clientFactory := clusterclient.NewFactory()
	kubeConifg, err := GetAdminConfig(cm)
	if err != nil {
		return nil, err
	}

	config := api.Convert_KubeConfig_To_Config(kubeConifg)
	data, err := clientcmd.Write(*config)
	if err != nil {
		return nil, err
	}
	bootstrapClient, err := clientFactory.NewClientFromKubeconfig(string(data))
	if err != nil {
		return nil, fmt.Errorf("unable to create bootstrap client: %v", err)
	}
	return bootstrapClient, nil
}

func GetKubernetesClient(cm Interface, cluster *api.Cluster) (kubernetes.Interface, error) {
	kubeConifg, err := GetAdminConfig(cm)
	if err != nil {
		return nil, err
	}

	config := api.NewRestConfig(kubeConifg)

	return kubernetes.NewForConfig(config)
}

func GetLeaderMachine(cluster *v1beta1.Cluster) (*clusterapi.Machine, error) {
	machine, err := store.StoreProvider.Machine(cluster.Name).Get(cluster.Name + "-master-0")
	if err != nil {
		return nil, err
	}
	return machine, nil
}
