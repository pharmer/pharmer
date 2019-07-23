package kube

import (
	"fmt"

	"github.com/appscode/go/log"
	"github.com/appscode/go/wait"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"gomodules.xyz/cert"
	semver "gomodules.xyz/version"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	kubeadmconsts "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	api "pharmer.dev/pharmer/apis/v1beta1"
	"pharmer.dev/pharmer/cloud/utils/certificates"
	"pharmer.dev/pharmer/store"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/clusterdeployer/clusterclient"
	"sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset"
)

func NewRestConfig(certStore store.CertificateStore, clusterEndpoint string) (*rest.Config, error) {
	caCert, caKey, err := certStore.Get(kubeadmconsts.CACertAndKeyBaseName)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get ca-certs")
	}

	adminCert, adminKey, err := certStore.Get("admin")
	if err != nil {
		adminCert, adminKey, err = certificates.CreateAdminCertificate(caCert, caKey)
		if err != nil {
			return nil, err
		}
	}

	cfg := &rest.Config{
		Host: clusterEndpoint,
		TLSClientConfig: rest.TLSClientConfig{
			CAData:   cert.EncodeCertPEM(caCert),
			CertData: cert.EncodeCertPEM(adminCert),
			KeyData:  cert.EncodePrivateKeyPEM(adminKey),
		},
	}

	return cfg, nil
}

func NewRestConfigFromKubeConfig(in *api.KubeConfig) *rest.Config {
	out := &rest.Config{
		Host: in.Cluster.Server,
		TLSClientConfig: rest.TLSClientConfig{
			CAData: append([]byte(nil), in.Cluster.CertificateAuthorityData...),
		},
	}
	if in.AuthInfo.Token == "" {
		out.TLSClientConfig.CertData = append([]byte(nil), in.AuthInfo.ClientCertificateData...)
		out.TLSClientConfig.KeyData = append([]byte(nil), in.AuthInfo.ClientKeyData...)
	} else {
		out.BearerToken = in.AuthInfo.Token
	}
	return out
}

//func GetKubernetesClient(s *cloud.Scope) (kubernetes.Interface, error) {
//	kubeConifg, err := GetAdminConfig(s)
//	if err != nil {
//		return nil, err
//	}
//
//	config := api.NewRestConfigFromKubeConfig(kubeConifg)
//
//	return kubernetes.NewForConfig(config)
//}

// WARNING:
// Returned KubeClient uses admin client cert. This should only be used for Cluster provisioning operations.
func NewAdminClient(certStore store.CertificateStore, clusterEndpoint string) (kubernetes.Interface, error) {
	cfg, err := NewRestConfig(certStore, clusterEndpoint)
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

func GetClusterAPIClient(certStore store.CertificateStore, clusterEndpoint string) (clientset.Interface, error) {
	conf, err := NewRestConfig(certStore, clusterEndpoint)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get rest config")
	}
	return clientset.NewForConfig(conf)
}

func waitForReadyAPIServer(log logr.Logger, client kubernetes.Interface) error {
	log.Info("waiting for kubernetes apiserver to be ready")
	attempt := 0
	return wait.PollImmediate(api.RetryInterval, api.RetryTimeout, func() (bool, error) {
		attempt++
		log.V(4).Info("probing kubernetes apiserver", "attempt", attempt)

		_, err := client.CoreV1().Pods(corev1.NamespaceAll).List(metav1.ListOptions{})
		if err != nil {
			log.V(6).Info("failed to list pods", "error", err.Error())
			return false, nil
		}

		log.Info("kubernetes apiserver is ready")
		return true, nil
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

func waitForReadyComponents(log logr.Logger, client kubernetes.Interface) error {
	attempt := 0
	log.Info("waiting for kubernetes components to be healthy")
	return wait.PollImmediate(api.RetryInterval, api.RetryTimeout, func() (bool, error) {
		attempt++
		log.V(4).Info("probing components", "Attempt", attempt)

		resp, err := client.CoreV1().ComponentStatuses().List(metav1.ListOptions{
			LabelSelector: labels.Everything().String(),
		})
		if err != nil {
			log.V(6).Info("error listing component status", "error", err.Error())
			return false, nil
		}
		for _, status := range resp.Items {
			for _, cond := range status.Conditions {
				if cond.Type == corev1.ComponentHealthy && cond.Status != corev1.ConditionTrue {
					log.V(4).Info("", "component", status.Name, "condition", cond.Type, "status", cond.Status)
					return false, nil
				}
			}
		}
		log.Info("kubernetes components are healthy")
		return true, nil
	})
}

func WaitForReadyMaster(log logr.Logger, client kubernetes.Interface) error {
	err := waitForReadyAPIServer(log, client)
	if err != nil {
		return err
	}
	return waitForReadyComponents(log, client)
}
