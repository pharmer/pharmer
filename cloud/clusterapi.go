package cloud

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/appscode/go/log"
	"github.com/appscode/go/wait"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud/utils/certificates"
	"github.com/pharmer/pharmer/cloud/utils/kube"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	"gomodules.xyz/cert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	kubeadmconsts "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/clusterdeployer/clusterclient"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/phases"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset"
)

const (
	MachineControllerImage = "pharmer/machine-controller:0.3.0"
)

type ClusterAPI struct {
	Interface

	namespace string
	token     string

	kubeClient       kubernetes.Interface
	clusterapiClient clientset.Interface
	bootstrapClient  clusterclient.Client

	externalController bool
}

type ApiServerTemplate struct {
	ClusterName         string
	Provider            string
	ControllerNamespace string
	ControllerImage     string
}

func NewClusterApi(cm Interface, namespace string, kc kubernetes.Interface) (*ClusterAPI, error) {
	token, err := kube.GetExistingKubeadmToken(kc, kubeadmconsts.DefaultTokenDuration)
	if err != nil {
		return nil, err
	}

	cluster := cm.GetCluster()
	bc, err := GetBooststrapClient(cm, cluster)
	if err != nil {
		return nil, err
	}
	clusterClient, err := GetClusterAPIClient(cm.GetCaCertPair(), cluster)
	if err != nil {
		return nil, err
	}

	return &ClusterAPI{
		Interface: cm,
		namespace: namespace,

		kubeClient:       kc,
		clusterapiClient: clusterClient,
		token:            token,
		bootstrapClient:  bc,
	}, nil
}

func GetClusterAPIClient(caCert *certificates.CertKeyPair, cluster *api.Cluster) (clientset.Interface, error) {
	conf, err := kube.NewRestConfig(caCert, cluster)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get rest config")
	}
	return clientset.NewForConfig(conf)
}

func (ca *ClusterAPI) Apply(controllerManager string) error {
	log.Infof("Deploying the addon apiserver and controller manager...")
	if err := ca.CreateMachineController(controllerManager); err != nil {
		return errors.Wrap(err, "can't create machine controller")
	}

	cluster := ca.GetCluster()
	if err := phases.ApplyCluster(ca.bootstrapClient, &cluster.Spec.ClusterAPI); err != nil && !api.ErrAlreadyExist(err) {
		return errors.Wrap(err, "failed to add Cluster")
	}
	namespace := cluster.Spec.ClusterAPI.Namespace
	if namespace == "" {
		namespace = ca.bootstrapClient.GetContextNamespace()
	}

	c, err := ca.clusterapiClient.ClusterV1alpha1().Clusters(namespace).Get(cluster.Name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update Cluster provider status")
	}

	c.Status = cluster.Spec.ClusterAPI.Status
	if _, err := ca.clusterapiClient.ClusterV1alpha1().Clusters(namespace).UpdateStatus(c); err != nil && !api.ErrObjectModified(err) {
		return errors.Wrap(err, "failed to update Cluster")
	}

	if err := ca.updateProviderStatus(); err != nil {
		log.Infoln(err)
		return errors.Wrap(err, "failed to update provider status")
	}

	masterMachine, err := GetLeaderMachine(cluster)
	if err != nil {
		return errors.Wrap(err, "failed to get leader machine")
	}

	masterMachine.Annotations = make(map[string]string)
	masterMachine.Annotations[InstanceStatusAnnotationKey] = ""

	log.Infof("Adding master machines...")
	err = phases.ApplyMachines(ca.bootstrapClient, namespace, []*clusterv1.Machine{masterMachine})
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
	pharmerCluster := ca.GetCluster()
	return wait.PollImmediate(api.RetryInterval, api.RetryTimeout, func() (bool, error) {
		cluster, err := ca.clusterapiClient.ClusterV1alpha1().Clusters(pharmerCluster.Spec.ClusterAPI.Namespace).Get(pharmerCluster.Name, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}
		if cluster.Status.ProviderStatus != nil {
			pharmerCluster.Spec.ClusterAPI.Status.ProviderStatus = cluster.Status.ProviderStatus
			if _, err := store.StoreProvider.Clusters().Update(pharmerCluster); err != nil {
				log.Info(err)
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
	log.Infoln("creating pharmer secret")
	if err := ca.CreatePharmerSecret(); err != nil {
		return err
	}

	log.Infoln("creating apiserver and controller")
	if err := ca.CreateApiServerAndController(controllerManager); err != nil && !api.ErrObjectModified(err) {
		return err
	}
	return nil
}

func (ca *ClusterAPI) CreatePharmerSecret() error {
	cluster := ca.GetCluster()
	providerConfig := cluster.ClusterConfig()

	cred, err := store.StoreProvider.Credentials().Get(cluster.ClusterConfig().CredentialName)
	if err != nil {
		return err
	}
	credData, err := json.MarshalIndent(cred, "", "  ")
	if err != nil {
		return err
	}

	if err = kube.CreateNamespace(ca.kubeClient, ca.namespace); err != nil {
		return err
	}

	if err = kube.CreateSecret(ca.kubeClient, "pharmer-cred", ca.namespace, map[string][]byte{
		fmt.Sprintf("%v.json", cluster.ClusterConfig().CredentialName): credData,
	}); err != nil {
		return err
	}

	if !ca.externalController {
		err := kube.CreateCredentialSecret(ca.kubeClient, cluster, ca.namespace)
		if err != nil {
			return err
		}
	}

	clusterData, err := json.MarshalIndent(cluster, "", "  ")
	if err != nil {
		return err
	}
	if err = kube.CreateSecret(ca.kubeClient, "pharmer-cluster", ca.namespace, map[string][]byte{
		fmt.Sprintf("%v.json", cluster.Name): clusterData,
	}); err != nil {
		return err
	}

	publicKey, privateKey, err := store.StoreProvider.SSHKeys(cluster.Name).Get(cluster.ClusterConfig().Cloud.SSHKeyName)
	if err != nil {
		return err
	}
	if err = kube.CreateSecret(ca.kubeClient, "pharmer-ssh", ca.namespace, map[string][]byte{
		fmt.Sprintf("id_%v", providerConfig.Cloud.SSHKeyName):     privateKey,
		fmt.Sprintf("id_%v.pub", providerConfig.Cloud.SSHKeyName): publicKey,
	}); err != nil {
		return err
	}

	certs := ca.GetPharmerCertificates()
	if err = kube.CreateSecret(ca.kubeClient, "pharmer-certificate", ca.namespace, map[string][]byte{
		"ca.crt":             cert.EncodeCertPEM(certs.CACert.Cert),
		"ca.key":             cert.EncodePrivateKeyPEM(certs.CACert.Key),
		"front-proxy-ca.crt": cert.EncodeCertPEM(certs.FrontProxyCACert.Cert),
		"front-proxy-ca.key": cert.EncodePrivateKeyPEM(certs.FrontProxyCACert.Key),
		"sa.crt":             cert.EncodeCertPEM(certs.ServiceAccountCert.Cert),
		"sa.key":             cert.EncodePrivateKeyPEM(certs.ServiceAccountCert.Key),
	}); err != nil {
		return err
	}

	if err = kube.CreateSecret(ca.kubeClient, "pharmer-etcd", ca.namespace, map[string][]byte{
		"ca.crt": cert.EncodeCertPEM(certs.EtcdCACert.Cert),
		"ca.key": cert.EncodePrivateKeyPEM(certs.EtcdCACert.Key),
	}); err != nil {
		return err
	}

	return nil
}

func (ca *ClusterAPI) CreateApiServerAndController(controllerManager string) error {
	tmpl, err := template.New("config").Parse(controllerManager)
	if err != nil {
		return err
	}
	cluster := ca.GetCluster()
	var tmplBuf bytes.Buffer
	err = tmpl.Execute(&tmplBuf, ApiServerTemplate{
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
