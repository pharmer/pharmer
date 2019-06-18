package kube

import (
	"fmt"

	"github.com/appscode/go/log"
	"github.com/appscode/go/wait"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud/utils/certificates"
	"github.com/pkg/errors"
	"gomodules.xyz/cert"
	semver "gomodules.xyz/version"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/clusterdeployer/clusterclient"
	"sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset"
)

func NewRestConfig(caCertPair *certificates.CertKeyPair, cluster *api.Cluster) (*rest.Config, error) {
	adminCert, adminKey, err := certificates.CreateAdminCertificate(caCertPair.Cert, caCertPair.Key)
	if err != nil {
		return nil, err
	}

	host := cluster.APIServerURL()
	if host == "" {
		return nil, errors.Errorf("failed to detect api server url for Cluster %s", cluster.Name)
	}

	cfg := &rest.Config{
		Host: host,
		TLSClientConfig: rest.TLSClientConfig{
			CAData:   cert.EncodeCertPEM(caCertPair.Cert),
			CertData: cert.EncodeCertPEM(adminCert),
			KeyData:  cert.EncodePrivateKeyPEM(adminKey),
		},
	}

	return cfg, nil
}

//func GetKubernetesClient(s *cloud.Scope) (kubernetes.Interface, error) {
//	kubeConifg, err := GetAdminConfig(s)
//	if err != nil {
//		return nil, err
//	}
//
//	config := api.NewRestConfig(kubeConifg)
//
//	return kubernetes.NewForConfig(config)
//}

// WARNING:
// Returned KubeClient uses admin client cert. This should only be used for Cluster provisioning operations.
func NewAdminClient(caCertPair *certificates.CertKeyPair, cluster *api.Cluster) (kubernetes.Interface, error) {
	cfg, err := NewRestConfig(caCertPair, cluster)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(cfg)
}

func GetBooststrapClient(cluster *api.Cluster, caCert *certificates.CertKeyPair) (clusterclient.Client, error) {
	clientFactory := clusterclient.NewFactory()
	kubeConifg, err := GetAdminConfig(cluster, caCert)
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

func GetAdminConfig(cluster *api.Cluster, caCertPair *certificates.CertKeyPair) (*api.KubeConfig, error) {
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

func GetClusterAPIClient(caCert *certificates.CertKeyPair, cluster *api.Cluster) (clientset.Interface, error) {
	conf, err := NewRestConfig(caCert, cluster)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get rest config")
	}
	return clientset.NewForConfig(conf)
}

func waitForReadyAPIServer(client kubernetes.Interface) error {
	attempt := 0
	return wait.PollImmediate(api.RetryInterval, api.RetryTimeout, func() (bool, error) {
		attempt++
		log.Infof("Attempt %v: Probing Kubernetes api server ...", attempt)

		_, err := client.CoreV1().Pods(corev1.NamespaceAll).List(metav1.ListOptions{})
		return err == nil, nil
	})
}

func WaitForReadyMasterVersion(client kubernetes.Interface, desiredVersion *semver.Version) error {
	attempt := 0
	var masterInstance *corev1.Node
	return wait.PollImmediate(api.RetryInterval, api.RetryTimeout, func() (bool, error) {
		attempt++
		log.Infof("Attempt %v: Upgrading to version %v ...", attempt, desiredVersion.String())
		masterInstances, err := client.CoreV1().Nodes().List(metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(map[string]string{
				api.RoleMasterKey: "",
			}).String(),
		})
		if err != nil {
			return false, nil
		}

		switch len(masterInstances.Items) {
		case 1:
			masterInstance = &masterInstances.Items[0]
		case 0:
			return false, nil
		default:
			return false, errors.Errorf("multiple master found")
		}

		currentVersion, _ := semver.NewVersion(masterInstance.Status.NodeInfo.KubeletVersion)

		if currentVersion.Equal(desiredVersion) {
			return true, nil
		}
		return false, nil

	})

}

func waitForReadyComponents(client kubernetes.Interface) error {
	attempt := 0
	return wait.PollImmediate(api.RetryInterval, api.RetryTimeout, func() (bool, error) {
		attempt++
		log.Infof("Attempt %v: Probing components ...", attempt)

		resp, err := client.CoreV1().ComponentStatuses().List(metav1.ListOptions{
			LabelSelector: labels.Everything().String(),
		})
		if err != nil {
			return false, nil
		}
		for _, status := range resp.Items {
			for _, cond := range status.Conditions {
				if cond.Type == corev1.ComponentHealthy && cond.Status != corev1.ConditionTrue {
					log.Infof("Component %v is in condition %v with status %v", status.Name, cond.Type, cond.Status)
					return false, nil
				}
			}
		}
		return true, nil
	})
}

func WaitForReadyMaster(client kubernetes.Interface) error {
	err := waitForReadyAPIServer(client)
	if err != nil {
		return err
	}
	return waitForReadyComponents(client)
}