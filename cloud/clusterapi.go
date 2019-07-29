package cloud

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/appscode/go/wait"
	"github.com/pkg/errors"
	"gomodules.xyz/cert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeadmconsts "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	api "pharmer.dev/pharmer/apis/v1alpha1"
	"pharmer.dev/pharmer/cloud/utils/kube"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/clusterdeployer/clusterclient"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/phases"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset"
)

const (
	MachineControllerImage = "pharmer/machine-controller:0.3.0"
)

type ClusterAPI struct {
	*Scope

	namespace string
	token     string

	clusterapiClient clientset.Interface
	bootstrapClient  clusterclient.Client

	externalController bool
}

type apiServerTemplate struct {
	ClusterName         string
	Provider            string
	ControllerNamespace string
	ControllerImage     string
}

func NewClusterAPI(s *Scope, namespace string) (*ClusterAPI, error) {
	token, err := kube.GetExistingKubeadmToken(s.AdminClient, kubeadmconsts.DefaultTokenDuration)
	if err != nil {
		return nil, err
	}

	bc, err := kube.GetBooststrapClient(s.Cluster, s.GetCaCertPair())
	if err != nil {
		s.Logger.Error(err, "failed to get bootstrap client")
		return nil, err
	}

	clusterEndpoint := s.Cluster.APIServerURL()
	if clusterEndpoint == "" {
		return nil, errors.Errorf("failed to detect api server url for Cluster %s", s.Cluster.Name)
	}

	clusterClient, err := kube.GetClusterAPIClient(s.StoreProvider.Certificates(s.Cluster.Name), clusterEndpoint)
	if err != nil {
		s.Logger.Error(err, "failed to get clusterAPI clients")
		return nil, err
	}

	return &ClusterAPI{
		Scope:            s,
		namespace:        namespace,
		clusterapiClient: clusterClient,
		token:            token,
		bootstrapClient:  bc,
	}, nil
}

func (ca *ClusterAPI) Apply(controllerManager string) error {
	log := ca.Logger
	log.Info("Deploying the addon apiserver and controller manager")
	if err := ca.CreateMachineController(controllerManager); err != nil &&
		!strings.Contains(err.Error(), "Already Exists,  Ignoring") {
		return errors.Wrap(err, "can't create machine controller")
	}

	cluster := ca.Cluster
	if err := phases.ApplyCluster(ca.bootstrapClient, &cluster.Spec.ClusterAPI); err != nil && !api.ErrAlreadyExist(err) {
		log.Error(err, "failed to add cluster")
		return err
	}
	namespace := cluster.Spec.ClusterAPI.Namespace
	if namespace == "" {
		namespace = ca.bootstrapClient.GetContextNamespace()
	}

	c, err := ca.clusterapiClient.ClusterV1alpha1().Clusters(namespace).Get(cluster.Name, metav1.GetOptions{})
	if err != nil {
		log.Error(err, "failed to update cluster provider status")
		return err
	}

	c.Status = cluster.Spec.ClusterAPI.Status
	if _, err := ca.clusterapiClient.ClusterV1alpha1().Clusters(namespace).UpdateStatus(c); err != nil && !api.ErrObjectModified(err) {
		log.Error(err, "failed to update Cluster")
		return err
	}

	if err := ca.updateProviderStatus(); err != nil {
		log.Error(err, "failder ot update provider status")
		return err
	}

	masterMachine, err := getLeaderMachine(ca.StoreProvider.Machine(ca.Cluster.Name), ca.Cluster.Name)
	if err != nil {
		log.Error(err, "failed to get leader machine")
		return err
	}

	log.Info("Adding master machines...")
	_, err = ca.clusterapiClient.ClusterV1alpha1().Machines(namespace).Create(masterMachine)
	if err != nil && !api.ErrAlreadyExist(err) && !api.ErrObjectModified(err) {
		return errors.Wrap(err, "failed to add master machine")
	}

	// get the machine object and update the provider status field
	err = ca.updateMachineStatus(namespace, masterMachine)
	if err != nil {
		return errors.Wrap(err, "failed to update machine status")
	}

	return nil
}

func (ca *ClusterAPI) updateProviderStatus() error {
	log := ca.Logger

	pharmerCluster := ca.Cluster
	return wait.PollImmediate(api.RetryInterval, api.RetryTimeout, func() (bool, error) {
		cluster, err := ca.clusterapiClient.ClusterV1alpha1().Clusters(pharmerCluster.Spec.ClusterAPI.Namespace).Get(pharmerCluster.Name, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}
		if cluster.Status.ProviderStatus != nil {
			pharmerCluster.Spec.ClusterAPI.Status.ProviderStatus = cluster.Status.ProviderStatus
			if _, err := ca.StoreProvider.Clusters().Update(pharmerCluster); err != nil {
				log.Error(err, "failed to update cluster status")
				return false, nil
			}
			return true, nil
		}
		return false, nil
	})
}

func (ca *ClusterAPI) updateMachineStatus(namespace string, masterMachine *clusterv1.Machine) error {
	return wait.PollImmediate(api.RetryInterval, api.RetryTimeout, func() (bool, error) {
		m, err := ca.clusterapiClient.ClusterV1alpha1().Machines(namespace).Get(masterMachine.Name, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}
		m.Status.ProviderStatus = masterMachine.Status.ProviderStatus
		if _, err := ca.clusterapiClient.ClusterV1alpha1().Machines(namespace).UpdateStatus(m); err != nil {
			return false, nil
		}
		return true, nil
	})
}

func (ca *ClusterAPI) CreateMachineController(controllerManager string) error {
	log := ca.Logger
	log.Info("creating pharmer secret")
	if err := ca.CreatePharmerSecret(); err != nil {
		return err
	}

	log.Info("creating apiserver and controller")
	if err := ca.CreateAPIServerAndController(controllerManager); err != nil && !api.ErrObjectModified(err) {
		return err
	}
	return nil
}

func (ca *ClusterAPI) CreatePharmerSecret() error {
	if ca.externalController {
		return nil
	}

	cluster := ca.Cluster
	providerConfig := cluster.ClusterConfig()

	cred, err := ca.StoreProvider.Credentials().Get(cluster.ClusterConfig().CredentialName)
	if err != nil {
		return err
	}
	credData, err := json.MarshalIndent(cred, "", "  ")
	if err != nil {
		return err
	}

	if err = kube.CreateNamespace(ca.AdminClient, ca.namespace); err != nil {
		return err
	}

	if err = kube.CreateSecret(ca.AdminClient, "pharmer-cred", ca.namespace, map[string][]byte{
		fmt.Sprintf("%v.json", cluster.ClusterConfig().CredentialName): credData,
	}); err != nil {
		return err
	}

	//	err := kube.CreateCredentialSecret(ca.AdminClient, cluster.CloudProvider(), ca.namespace, ca.CredentialData)
	//	if err != nil {
	//		return err
	//	}

	clusterData, err := json.MarshalIndent(cluster, "", "  ")
	if err != nil {
		return err
	}
	if err = kube.CreateSecret(ca.AdminClient, "pharmer-cluster", ca.namespace, map[string][]byte{
		fmt.Sprintf("%v.json", cluster.Name): clusterData,
	}); err != nil {
		return err
	}

	publicKey, privateKey, err := ca.StoreProvider.SSHKeys(cluster.Name).Get(cluster.ClusterConfig().Cloud.SSHKeyName)
	if err != nil {
		return err
	}
	if err = kube.CreateSecret(ca.AdminClient, "pharmer-ssh", ca.namespace, map[string][]byte{
		fmt.Sprintf("id_%v", providerConfig.Cloud.SSHKeyName):     privateKey,
		fmt.Sprintf("id_%v.pub", providerConfig.Cloud.SSHKeyName): publicKey,
	}); err != nil {
		return err
	}

	certs := ca.GetCertificates()
	if err = kube.CreateSecret(ca.AdminClient, "pharmer-certificate", ca.namespace, map[string][]byte{
		"ca.crt":             cert.EncodeCertPEM(certs.CACert.Cert),
		"ca.key":             cert.EncodePrivateKeyPEM(certs.CACert.Key),
		"front-proxy-ca.crt": cert.EncodeCertPEM(certs.FrontProxyCACert.Cert),
		"front-proxy-ca.key": cert.EncodePrivateKeyPEM(certs.FrontProxyCACert.Key),
		"sa.crt":             cert.EncodeCertPEM(certs.ServiceAccountCert.Cert),
		"sa.key":             cert.EncodePrivateKeyPEM(certs.ServiceAccountCert.Key),
	}); err != nil {
		return err
	}

	if err = kube.CreateSecret(ca.AdminClient, "pharmer-etcd", ca.namespace, map[string][]byte{
		"ca.crt": cert.EncodeCertPEM(certs.EtcdCACert.Cert),
		"ca.key": cert.EncodePrivateKeyPEM(certs.EtcdCACert.Key),
	}); err != nil {
		return err
	}

	return nil
}

func (ca *ClusterAPI) CreateAPIServerAndController(controllerManager string) error {
	tmpl, err := template.New("config").Parse(controllerManager)
	if err != nil {
		return err
	}
	cluster := ca.Cluster
	var tmplBuf bytes.Buffer
	err = tmpl.Execute(&tmplBuf, apiServerTemplate{
		ClusterName:         cluster.Name,
		Provider:            cluster.ClusterConfig().Cloud.CloudProvider,
		ControllerNamespace: ca.namespace,
		ControllerImage:     MachineControllerImage,
	})
	if err != nil {
		return err
	}

	return ca.bootstrapClient.Apply(tmplBuf.String())
}
