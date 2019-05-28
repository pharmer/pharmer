package cloud

import (
	"context"
	"fmt"
	"time"

	"github.com/appscode/go/log"
	"github.com/pharmer/pharmer/apis/v1beta1"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud/cmds/options"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	kubeadmconst "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/clusterdeployer/clusterclient"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/util"
)

var managedProviders = sets.NewString("aks", "gke", "eks", "dokube")

func List(ctx context.Context, opts metav1.ListOptions, owner string) ([]*api.Cluster, error) {
	return Store(ctx).Clusters().List(opts)
}

func Get(name string) (*api.Cluster, error) {
	return store.StoreProvider.Clusters().Get(name)
}

func getPharmerCerts(cluster *api.Cluster) (*api.PharmerCertificates, error) {
	pharmerCerts := &api.PharmerCertificates{}

	cert, key, err := LoadCACertificates(cluster.Name, kubeadmconst.CACertAndKeyBaseName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load ca certs")
	}
	pharmerCerts.CACert = api.CertKeyPair{
		Cert: cert,
		Key:  key,
	}

	cert, key, err = LoadCACertificates(cluster.Name, kubeadmconst.FrontProxyCACertAndKeyBaseName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load fpca certs")
	}
	pharmerCerts.FrontProxyCACert = api.CertKeyPair{
		Cert: cert,
		Key:  key,
	}

	cert, key, err = LoadCACertificates(cluster.Name, kubeadmconst.ServiceAccountKeyBaseName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load sa keys")
	}
	pharmerCerts.ServiceAccountCert = api.CertKeyPair{
		Cert: cert,
		Key:  key,
	}

	cert, key, err = LoadCACertificates(cluster.Name, kubeadmconst.EtcdCACertAndKeyBaseName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load etcd-ca keys")
	}
	pharmerCerts.EtcdCACert = api.CertKeyPair{
		Cert: cert,
		Key:  key,
	}

	pubKey, privKey, err := LoadSSHKey(cluster)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load ssh keys")
	}
	pharmerCerts.SSHKey = api.SSHKey{
		PublicKey:  pubKey,
		PrivateKey: privKey,
	}

	return pharmerCerts, nil
}

func createPharmerCerts(cluster *api.Cluster) (*api.PharmerCertificates, error) {
	pharmerCerts := &api.PharmerCertificates{}

	cert, key, err := CreateCACertificates(store.StoreProvider, cluster.Name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create ca certificates")
	}
	pharmerCerts.CACert = api.CertKeyPair{
		Cert: cert,
		Key:  key,
	}

	cert, key, err = CreateFrontProxyCACertificates(store.StoreProvider, cluster.Name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create fpca certificates")
	}
	pharmerCerts.FrontProxyCACert = api.CertKeyPair{
		Cert: cert,
		Key:  key,
	}

	cert, key, err = CreateSACertificate(store.StoreProvider, cluster.Name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create sa certificates")
	}
	pharmerCerts.ServiceAccountCert = api.CertKeyPair{
		Cert: cert,
		Key:  key,
	}

	cert, key, err = CreateEtcdCACertificate(store.StoreProvider, cluster.Name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create etcd-ca certificates")
	}
	pharmerCerts.EtcdCACert = api.CertKeyPair{
		Cert: cert,
		Key:  key,
	}

	pubKey, privKey, err := CreateSSHKey(store.StoreProvider, cluster)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create ssh keys")
	}
	pharmerCerts.SSHKey = api.SSHKey{
		PublicKey:  pubKey,
		PrivateKey: privKey,
	}

	return pharmerCerts, nil
}

func Delete(ctx context.Context, name string, owner string) (*api.Cluster, error) {
	if name == "" {
		return nil, errors.New("missing cluster name")
	}

	cluster, err := Store(ctx).Clusters().Get(name)
	if err != nil {
		return nil, errors.Errorf("cluster `%s` does not exist. Reason: %v", name, err)
	}
	cluster.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	cluster.Status.Phase = api.ClusterDeleting

	return Store(ctx).Clusters().Update(cluster)
}

func DeleteMachineSet(ctx context.Context, clusterName, setName, owner string) error {
	if clusterName == "" {
		return errors.New("missing cluster name")
	}
	if setName == "" {
		return errors.New("missing machineset name")
	}

	mSet, err := Store(ctx).MachineSet(clusterName).Get(setName)
	if err != nil {
		return errors.Errorf(`machinset not found in pharmer db, try using kubectl`)
	}
	tm := metav1.Now()
	mSet.DeletionTimestamp = &tm
	_, err = Store(ctx).MachineSet(clusterName).Update(mSet)
	return err
}

func GetSSHConfig(ctx context.Context, owner, nodeName string, cluster *api.Cluster) (*api.SSHConfig, error) {
	//var err error
	//ctx, err = LoadCACertificates(ctx, cluster, owner)
	//if err != nil {
	//	return nil, err
	//}
	//client, err := NewAdminClient(ctx, cluster)
	//if err != nil {
	//	return nil, err
	//}
	//node, err := client.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
	//if err != nil {
	//	return nil, err
	//}
	//ctx, err = LoadSSHKey(ctx, cluster, owner)
	//if err != nil {
	//	return nil, err
	//}
	//
	//cm, err := GetCloudManager(cluster.ClusterConfig().Cloud.CloudProvider)
	//if err != nil {
	//	return nil, err
	//}
	//return cm.GetSSHConfig(cluster, node)
	return nil, nil
}

func GetAdminConfig(ctx context.Context, cluster *api.Cluster, owner string) (*api.KubeConfig, error) {
	//if managedProviders.Has(cluster.ClusterConfig().Cloud.CloudProvider) {
	//	cm, err := GetCloudManager(cluster.ClusterConfig().Cloud.CloudProvider)
	//	if err != nil {
	//		return nil, err
	//	}
	//	cm.SetOwner(owner)
	//	return cm.GetKubeConfig(cluster)
	//}
	//var err error
	//ctx, err = LoadCACertificates(ctx, cluster, owner)
	//if err != nil {
	//	return nil, err
	//}
	//adminCert, adminKey, err := CreateAdminCertificate(ctx)
	//if err != nil {
	//	return nil, err
	//}
	//
	//var (
	//	clusterName = fmt.Sprintf("%s.pharmer", cluster.Name)
	//	userName    = fmt.Sprintf("cluster-admin@%s.pharmer", cluster.Name)
	//	ctxName     = fmt.Sprintf("cluster-admin@%s.pharmer", cluster.Name)
	//)
	//cfg := api.KubeConfig{
	//	TypeMeta: metav1.TypeMeta{
	//		APIVersion: "v1",
	//		Kind:       "KubeConfig",
	//	},
	//	Preferences: api.Preferences{
	//		Colors: true,
	//	},
	//	Cluster: api.NamedCluster{
	//		Name:                     clusterName,
	//		Server:                   cluster.APIServerURL(),
	//		CertificateAuthorityData: cert.EncodeCertPEM(CACert(ctx)),
	//	},
	//	AuthInfo: api.NamedAuthInfo{
	//		Name:                  userName,
	//		ClientCertificateData: cert.EncodeCertPEM(adminCert),
	//		ClientKeyData:         cert.EncodePrivateKeyPEM(adminKey),
	//	},
	//	Context: api.NamedContext{
	//		Name:     ctxName,
	//		Cluster:  clusterName,
	//		AuthInfo: userName,
	//	},
	//}
	//return &cfg, nil
	return nil, nil
}

func Apply(ctx context.Context, opts *options.ApplyConfig) ([]api.Action, error) {
	//if opts.ClusterName == "" {
	//	return nil, errors.New("missing cluster name")
	//}
	//
	//cluster, err := Store(ctx).Clusters().Get(opts.ClusterName)
	//if err != nil {
	//	return nil, errors.Errorf("cluster `%s` does not exist. Reason: %v", opts.ClusterName, err)
	//}
	//
	//cm, err := GetCloudManager(cluster.ClusterConfig().Cloud.CloudProvider)
	//if err != nil {
	//	return nil, err
	//}
	//cm.SetOwner(opts.Owner)
	//
	//return cm.Apply(cluster, opts.DryRun)
	return nil, nil
}

func CheckForUpdates(ctx context.Context, name, owner string) (string, error) {
	//if name == "" {
	//	return "", errors.New("missing cluster name")
	//}
	//
	//cluster, err := Store(ctx).Clusters().Get(name)
	//if err != nil {
	//	return "", errors.Errorf("cluster `%s` does not exist. Reason: %v", name, err)
	//}
	//if cluster.Status.Phase == "" {
	//	return "", errors.Errorf("cluster `%s` is in unknown phase", cluster.Name)
	//}
	//if cluster.Status.Phase != api.ClusterReady {
	//	return "", errors.Errorf("cluster `%s` is not ready", cluster.Name)
	//}
	//if cluster.Status.Phase == api.ClusterDeleted {
	//	return "", nil
	//}
	//if ctx, err = LoadCACertificates(ctx, cluster, owner); err != nil {
	//	return "", err
	//}
	//if ctx, err = LoadSSHKey(ctx, cluster, owner); err != nil {
	//	return "", err
	//}
	//kc, err := NewAdminClient(ctx, cluster)
	//if err != nil {
	//	return "", err
	//}
	//
	//upm := NewUpgradeManager(ctx, kc, cluster, owner)
	//upgrades, err := upm.GetAvailableUpgrades()
	//if err != nil {
	//	return "", err
	//}
	//upm.PrintAvailableUpgrades(upgrades)
	return "", nil
}

func UpdateSpec(ctx context.Context, cluster *api.Cluster, owner string) (*api.Cluster, error) {
	if cluster == nil {
		return nil, errors.New("missing cluster")
	} else if cluster.Name == "" {
		return nil, errors.New("missing cluster name")
	} else if cluster.ClusterConfig().KubernetesVersion == "" {
		return nil, errors.New("missing cluster version")
	}

	existing, err := Store(ctx).Clusters().Get(cluster.Name)
	if err != nil {
		return nil, errors.Errorf("cluster `%s` does not exist. Reason: %v", cluster.Name, err)
	}
	cluster.Status = existing.Status
	cluster.Generation = time.Now().UnixNano()

	return Store(ctx).Clusters().Update(cluster)
}

func GetBooststrapClient(ctx context.Context, cluster *api.Cluster, owner string) (clusterclient.Client, error) {
	clientFactory := clusterclient.NewFactory()
	kubeConifg, err := GetAdminConfig(ctx, cluster, owner)
	if err != nil {
		return nil, err
	}

	config := api.Convert_KubeConfig_To_Config(kubeConifg)
	data, err := clientcmd.Write(*config)
	bootstrapClient, err := clientFactory.NewClientFromKubeconfig(string(data))
	if err != nil {
		return nil, fmt.Errorf("unable to create bootstrap client: %v", err)
	}
	return bootstrapClient, nil
}

func GetKubernetesClient(ctx context.Context, cluster *api.Cluster, owner string) (kubernetes.Interface, error) {
	kubeConifg, err := GetAdminConfig(ctx, cluster, owner)
	if err != nil {
		return nil, err
	}

	config := api.NewRestConfig(kubeConifg)

	return kubernetes.NewForConfig(config)
}

func GetLeaderMachine(ctx context.Context, cluster *v1beta1.Cluster, owner string) (*clusterapi.Machine, error) {
	machine, err := Store(ctx).Machine(cluster.Name).Get(cluster.Name + "-master-0")
	if err != nil {
		return nil, err
	}
	return machine, nil
}

/*func GetSSHConfig(ctx context.Context, hostip string) *api.SSHConfig {
	return &api.SSHConfig{
		PrivateKey: SSHKey(ctx).PrivateKey,
		User:       "root",
		HostPort:   int32(22),
		HostIP: hostip,
	}
}*/

// DeleteAllWorkerMachines waits for all nodes to be deleted
func DeleteAllWorkerMachines(ctx context.Context, cluster *v1beta1.Cluster, owner string) error {
	log.Infof("Deleting non-controlplane machines")

	client, err := GetBooststrapClient(ctx, cluster, owner)
	if err != nil {
		return errors.Wrap(err, "failed to get clusterapi client")
	}

	err = deleteMachineDeployments(client)
	if err != nil {
		log.Infof("failed to delete machine deployments: %v", err)
	}

	err = deleteMachineSets(client)
	if err != nil {
		log.Infof("failed to delete machinesetes: %v", err)
	}

	err = deleteMachines(client)
	if err != nil {
		log.Infof("failed to delete machines: %v", err)
	}

	log.Infof("successfully deleted non-controlplane machines")
	return nil
}

// deletes machinedeployments in all namespaces
func deleteMachineDeployments(client clusterclient.Client) error {
	err := client.DeleteMachineDeployments(corev1.NamespaceAll)
	if err != nil {
		return err
	}

	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (done bool, err error) {
		deployList, err := client.GetMachineDeployments(corev1.NamespaceAll)
		if err != nil {
			log.Infof("failed to list machine deployments: %v", err)
			return false, nil
		}
		if len(deployList) == 0 {
			log.Infof("successfully deleted machine deployments")
			return true, nil
		}
		log.Infof("machine deployments are not deleted yet")
		return false, nil
	})
}

// deletes machinesets in all namespaces
func deleteMachineSets(client clusterclient.Client) error {
	err := client.DeleteMachineSets(corev1.NamespaceAll)
	if err != nil {
		return err
	}

	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (done bool, err error) {
		machineSetList, err := client.GetMachineSets(corev1.NamespaceAll)
		if err != nil {
			log.Infof("failed to list machine sets: %v", err)
			return false, nil
		}
		if len(machineSetList) == 0 {
			log.Infof("successfully deleted machinesets")
			return true, nil
		}
		log.Infof("machinesets are not deleted yet")
		return false, nil
	})
}

// deletes machines in all namespaces
func deleteMachines(client clusterclient.Client) error {
	// delete non-controlplane machines
	machineList, err := client.GetMachines(corev1.NamespaceAll)
	for _, machine := range machineList {
		if !util.IsControlPlaneMachine(machine) && machine.DeletionTimestamp == nil {
			err = client.ForceDeleteMachine(machine.Namespace, machine.Name)
			if err != nil {
				log.Infof("failed to delete machine %s in namespace %s", machine.Namespace, machine.Name)
			}
		}
	}

	// wait for machines to be deleted
	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (done bool, err error) {
		machineList, err := client.GetMachines(corev1.NamespaceAll)
		for _, machine := range machineList {
			if !util.IsControlPlaneMachine(machine) {
				log.Infof("machine %s in namespace %s is not deleted yet", machine.Name, machine.Namespace)
			}
		}

		log.Infof("successfully deleted non-controlplane machines")
		return true, nil
	})
}
