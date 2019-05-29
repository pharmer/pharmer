package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gomodules.xyz/cert"

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
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/util"
)

var managedProviders = sets.NewString("aks", "gke", "eks", "dokube")

func List(ctx context.Context, opts metav1.ListOptions, owner string) ([]*api.Cluster, error) {
	return Store(ctx).Clusters().List(opts)
}

func GetCluster(name string) (*api.Cluster, error) {
	return store.StoreProvider.Clusters().Get(name)
}

func getPharmerCerts(clusterName string) (*api.PharmerCertificates, error) {
	pharmerCerts := &api.PharmerCertificates{}

	cert, key, err := LoadCACertificates(clusterName, kubeadmconst.CACertAndKeyBaseName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load ca certs")
	}
	pharmerCerts.CACert = api.CertKeyPair{
		Cert: cert,
		Key:  key,
	}

	cert, key, err = LoadCACertificates(clusterName, kubeadmconst.FrontProxyCACertAndKeyBaseName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load fpca certs")
	}
	pharmerCerts.FrontProxyCACert = api.CertKeyPair{
		Cert: cert,
		Key:  key,
	}

	cert, key, err = LoadCACertificates(clusterName, kubeadmconst.ServiceAccountKeyBaseName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load sa keys")
	}
	pharmerCerts.ServiceAccountCert = api.CertKeyPair{
		Cert: cert,
		Key:  key,
	}

	cert, key, err = LoadCACertificates(clusterName, kubeadmconst.EtcdCACertAndKeyBaseName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load etcd-ca keys")
	}
	pharmerCerts.EtcdCACert = api.CertKeyPair{
		Cert: cert,
		Key:  key,
	}

	pubKey, privKey, err := LoadSSHKey(GenSSHKeyName(clusterName))
	if err != nil {
		return nil, errors.Wrap(err, "failed to load ssh keys")
	}
	pharmerCerts.SSHKey = api.SSHKey{
		PublicKey:  pubKey,
		PrivateKey: privKey,
	}

	return pharmerCerts, nil
}

func GenSSHKeyName(clusterName string) string {
	return clusterName + "-sshkey"
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

func GetAdminConfig(cm Interface, cluster *api.Cluster) (*api.KubeConfig, error) {
	if managedProviders.Has(cluster.ClusterConfig().Cloud.CloudProvider) {
		return cm.GetKubeConfig(cluster)
	}
	var err error

	caCertPair := cm.GetCaCertPair()
	adminCert, adminKey, err := CreateAdminCertificate(caCertPair.Cert, caCertPair.Key)
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

func Apply(opts *options.ApplyConfig) ([]api.Action, error) {
	dryRun := opts.DryRun
	if opts.ClusterName == "" {
		return nil, errors.New("missing cluster name")
	}

	cluster, err := store.StoreProvider.Clusters().Get(opts.ClusterName)
	if err != nil {
		return nil, errors.Errorf("cluster `%s` does not exist. Reason: %v", opts.ClusterName, err)
	}

	cm, err := GetCloudManager(cluster)
	if err != nil {
		return nil, err
	}

	var acts []api.Action

	if cluster.Status.Phase == "" {
		return nil, errors.Errorf("cluster `%s` is in unknown phase", cluster.Name)
	}
	if cluster.Status.Phase == api.ClusterDeleted {
		return nil, nil
	}

	if err = cm.PrepareCloud(); err != nil {
		return nil, err
	}

	if cluster.Status.Phase == api.ClusterUpgrading {
		return nil, errors.Errorf("cluster `%s` is upgrading. Retry after cluster returns to Ready state", cluster.Name)
	}
	if cluster.Status.Phase == api.ClusterReady {
		var kc kubernetes.Interface
		kc, err = CreateAdminClient(cm)
		if err != nil {
			return nil, err
		}
		if upgrade, err := NewKubeVersionGetter(kc, cluster).IsUpgradeRequested(); err != nil {
			return nil, err
		} else if upgrade {
			cluster.Status.Phase = api.ClusterUpgrading
			_, _ = store.StoreProvider.Clusters().UpdateStatus(cluster)
			return cm.ApplyUpgrade(dryRun)
		}
	}

	if cluster.Status.Phase == api.ClusterPending {
		a, err := ApplyCreate(dryRun, cm)
		if err != nil {
			return nil, err
		}
		acts = append(acts, a...)
	}

	if cluster.DeletionTimestamp != nil && cluster.Status.Phase != api.ClusterDeleted {
		machines, err := store.StoreProvider.MachineSet(cluster.Name).List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		var replicas int32 = 0
		for _, machine := range machines {
			machine.Spec.Replicas = &replicas
			_, err := store.StoreProvider.MachineSet(cluster.Name).Update(machine)
			if err != nil {
				return nil, err
			}
		}
	}

	{
		a, err := ApplyScale(cm)
		if err != nil {
			// ignore error if cluster is deleted
			if cluster.DeletionTimestamp != nil && cluster.Status.Phase != api.ClusterDeleted {
				log.Infoln(err)
			} else {
				return nil, err
			}
		}
		acts = append(acts, a...)
	}

	if cluster.DeletionTimestamp != nil && cluster.Status.Phase != api.ClusterDeleted {
		a, err := cm.ApplyDelete(dryRun)
		if err != nil {
			return nil, err
		}
		acts = append(acts, a...)
	}

	return acts, nil
}

func ApplyCreate(dryRun bool, cm Interface) (acts []api.Action, err error) {
	acts, leaderMachine, machines, err := cm.ApplyCreate(dryRun)
	if err != nil {
		return acts, err
	}

	cluster := cm.GetCluster()

	kc := cm.GetAdminClient()

	if err = WaitForReadyMaster(kc); err != nil {
		cluster.Status.Reason = err.Error()
		err = errors.Wrap(err, " error occurred while waiting for master")
		return acts, err
	}

	leaderMachine, err = store.StoreProvider.Machine(cluster.Name).UpdateStatus(leaderMachine)
	if err != nil {
		return
	}

	log.Infoln("Creating secret credential...")
	if err = CreateCredentialSecret(kc, cluster); err != nil {
		return acts, errors.Wrap(err, "Error creating ccm secret credentials")
	}

	conn := cm.GetConnector()
	var controllerManager string
	controllerManager, err = conn.GetControllerManager()

	if err != nil {
		return acts, errors.Wrap(err, "Error creating controller-manager")
	}

	ca, err := NewClusterApi(cm, cluster, "cloud-provider-system", kc, conn)
	if err != nil {
		return acts, errors.Wrap(err, "Error creating cluster-api components")
	}

	if err := ca.Apply(controllerManager); err != nil {
		return acts, err
	}

	log.Infof("Adding other master machines")
	client, err := GetClusterClient(cm, cluster)
	if err != nil {
		return nil, err
	}

	for _, m := range machines {
		if m.Name == leaderMachine.Name {
			continue
		}
		if _, err := client.ClusterV1alpha1().Machines(cluster.Spec.ClusterAPI.Namespace).Create(m); err != nil && !api.ErrObjectModified(err) {
			log.Infof("Error creating maching %q in namespace %q", m.Name, cluster.Spec.ClusterAPI.Namespace)
			return acts, err
		}
	}

	cluster.Status.Phase = api.ClusterReady
	if _, err = store.StoreProvider.Clusters().Update(cluster); err != nil {
		return
	}

	return acts, nil
}

func ApplyScale(cm Interface) (acts []api.Action, err error) {
	log.Infoln("Scaling Machine Sets")
	cluster := cm.GetCluster()
	var machineSets []*clusterv1.MachineSet
	var existingMachineSet []*clusterv1.MachineSet
	machineSets, err = store.StoreProvider.MachineSet(cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var bc clusterclient.Client
	bc, err = GetBooststrapClient(cm, cluster)
	if err != nil {
		return nil, err
	}

	var data []byte
	for _, machineSet := range machineSets {
		if machineSet.DeletionTimestamp != nil {
			machineSet.DeletionTimestamp = nil
			if data, err = json.Marshal(machineSet); err != nil {
				return nil, err
			}
			if err = bc.Delete(string(data)); err != nil {
				return nil, err
			}
			if err = store.StoreProvider.MachineSet(cluster.Name).Delete(machineSet.Name); err != nil {
				return nil, err
			}
		}

		if existingMachineSet, err = bc.GetMachineSets(bc.GetContextNamespace()); err != nil {
			return nil, err
		}

		if data, err = json.Marshal(machineSet); err != nil {
			return nil, err
		}
		found := false
		for _, ems := range existingMachineSet {
			if ems.Name == machineSet.Name {
				found = true
				if err = bc.Apply(string(data)); err != nil {
					return nil, err
				}
				break
			}
		}

		if !found {
			if err = bc.CreateMachineSets([]*clusterv1.MachineSet{machineSet}, bc.GetContextNamespace()); err != nil {
				return nil, err
			}
		}
	}

	_, err = store.StoreProvider.Clusters().UpdateStatus(cluster)
	if err != nil {
		return nil, err
	}
	_, err = store.StoreProvider.Clusters().Update(cluster)
	if err != nil {
		return nil, err
	}

	return acts, nil
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

func GetBooststrapClient(cm Interface, cluster *api.Cluster) (clusterclient.Client, error) {
	clientFactory := clusterclient.NewFactory()
	kubeConifg, err := GetAdminConfig(cm, cluster)
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

func GetKubernetesClient(cm Interface, cluster *api.Cluster, owner string) (kubernetes.Interface, error) {
	kubeConifg, err := GetAdminConfig(cm, cluster)
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

/*func GetSSHConfig(ctx context.Context, hostip string) *api.SSHConfig {
	return &api.SSHConfig{
		PrivateKey: SSHKey(ctx).PrivateKey,
		User:       "root",
		HostPort:   int32(22),
		HostIP: hostip,
	}
}*/

// DeleteAllWorkerMachines waits for all nodes to be deleted
func DeleteAllWorkerMachines(cm Interface, cluster *v1beta1.Cluster, owner string) error {
	log.Infof("Deleting non-controlplane machines")

	client, err := GetBooststrapClient(cm, cluster)
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

func CreateAdminClient(cm Interface) (kubernetes.Interface, error) {
	m := cm.GetMutex()
	m.Lock()
	defer m.Unlock()

	if kc := cm.GetAdminClient(); kc != nil {
		return kc, nil
	}
	kc, err := NewAdminClient(cm, cm.GetCluster())

	if err != nil {
		return nil, err
	}

	return kc, nil
}
