package cloud

import (
	"context"
	"fmt"
	"time"

	semver "github.com/appscode/go-version"
	stringz "github.com/appscode/go/strings"
	apiAlpha "github.com/pharmer/pharmer/apis/v1alpha1"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/cert"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	RetryInterval      = 5 * time.Second
	RetryTimeout       = 15 * time.Minute
	ServiceAccountNs   = "kube-system"
	ServiceAccountName = "default"
)

func NodeCount(machineSets []*clusterv1.MachineSet) int64 {
	count := int64(0)
	for _, machineSet := range machineSets {
		count += int64(*machineSet.Spec.Replicas)
	}
	return count
}

func FindMasterNodeGroup(nodeGroups []*apiAlpha.NodeGroup) (*apiAlpha.NodeGroup, error) {
	for _, ng := range nodeGroups {
		if ng.IsMaster() {
			return ng, nil
		}
	}
	return nil, ErrNoMasterNG
}

func IsNodeReady(node *core.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == core.NodeReady {
			return condition.Status == core.ConditionTrue
		}
	}

	return false
}

func NewRestConfig(ctx context.Context, cluster *api.Cluster) (*rest.Config, error) {
	adminCert, adminKey, err := CreateAdminCertificate(ctx)
	if err != nil {
		return nil, err
	}

	host := cluster.APIServerURL()
	if host == "" {
		return nil, errors.Errorf("failed to detect api server url for cluster %s", cluster.Name)
	}

	cfg := &rest.Config{
		Host: host,
		TLSClientConfig: rest.TLSClientConfig{
			CAData:   cert.EncodeCertPEM(CACert(ctx)),
			CertData: cert.EncodeCertPEM(adminCert),
			KeyData:  cert.EncodePrivateKeyPEM(adminKey),
		},
	}

	return cfg, nil
}

// WARNING:
// Returned KubeClient uses admin client cert. This should only be used for cluster provisioning operations.
func NewAdminClient(ctx context.Context, cluster *api.Cluster) (kubernetes.Interface, error) {
	cfg, err := NewRestConfig(ctx, cluster)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(cfg)
}

func waitForReadyAPIServer(ctx context.Context, client kubernetes.Interface) error {
	attempt := 0
	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		attempt++
		Logger(ctx).Infof("Attempt %v: Probing Kubernetes api server ...", attempt)

		_, err := client.CoreV1().Pods(core.NamespaceAll).List(metav1.ListOptions{})
		return err == nil, nil
	})
}

func WaitForReadyMasterVersion(ctx context.Context, client kubernetes.Interface, desiredVersion *semver.Version) error {
	attempt := 0
	var masterInstance *core.Node
	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		attempt++
		Logger(ctx).Infof("Attempt %v: Upgrading to version %v ...", attempt, desiredVersion.String())
		masterInstances, err := client.CoreV1().Nodes().List(metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(map[string]string{
				api.RoleMasterKey: "",
			}).String(),
		})
		if err != nil {
			return false, nil
		}
		if len(masterInstances.Items) == 1 {
			masterInstance = &masterInstances.Items[0]
		} else if len(masterInstances.Items) > 1 {
			return false, errors.Errorf("multiple master found")
		} else {
			return false, nil
		}

		currentVersion, _ := semver.NewVersion(masterInstance.Status.NodeInfo.KubeletVersion)

		if currentVersion.Equal(desiredVersion) {
			return true, nil
		}
		return false, nil

	})

}

func waitForReadyComponents(ctx context.Context, client kubernetes.Interface) error {
	attempt := 0
	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		attempt++
		Logger(ctx).Infof("Attempt %v: Probing components ...", attempt)

		resp, err := client.CoreV1().ComponentStatuses().List(metav1.ListOptions{
			LabelSelector: labels.Everything().String(),
		})
		if err != nil {
			return false, nil
		}
		for _, status := range resp.Items {
			for _, cond := range status.Conditions {
				if cond.Type == core.ComponentHealthy && cond.Status != core.ConditionTrue {
					Logger(ctx).Infof("Component %v is in condition %v with status %v", status.Name, cond.Type, cond.Status)
					return false, nil
				}
			}
		}
		return true, nil
	})
}

func WaitForReadyMaster(ctx context.Context, client kubernetes.Interface) error {
	err := waitForReadyAPIServer(ctx, client)
	if err != nil {
		return err
	}
	return waitForReadyComponents(ctx, client)
}

var restrictedNamespaces []string = []string{"appscode", "kube-system"}

func HasNoUserApps(client kubernetes.Interface) (bool, error) {
	pods, err := client.CoreV1().Pods(core.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		// If we can't connect to kube apiserver, then delete cluster.
		// Cluster probably failed to create.
		return true, nil
	}
	for _, pod := range pods.Items {
		if pod.Status.Phase == "Running" && !stringz.Contains(restrictedNamespaces, pod.Namespace) {
			return false, nil
		}
	}
	return true, nil
}

func DeleteLoadBalancers(client kubernetes.Interface) error {
	// Delete services with type = LoadBalancer
	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		svcs, err := client.CoreV1().Services("").List(metav1.ListOptions{})
		if err != nil {
			return false, nil
		}
		for _, svc := range svcs.Items {
			if svc.Spec.Type == core.ServiceTypeLoadBalancer {
				trueValue := true
				err = client.CoreV1().Services(svc.Namespace).Delete(svc.Name, &metav1.DeleteOptions{OrphanDependents: &trueValue})
				if err != nil {
					return false, nil
				}
			}
		}
		return true, nil
	})

}

func DeleteDyanamicVolumes(client kubernetes.Interface) error {
	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		pvcs, err := client.CoreV1().PersistentVolumeClaims("").List(metav1.ListOptions{})
		if err != nil {
			return false, nil
		}
		for _, pvc := range pvcs.Items {
			if pvc.Status.Phase == core.ClaimBound {
				for k, v := range pvc.Annotations {
					if (k == "volume.alpha.kubernetes.io/storage-class" ||
						k == "volume.beta.kubernetes.io/storage-class") && v != "" {
						trueValue := true
						err = client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Delete(pvc.Name, &metav1.DeleteOptions{OrphanDependents: &trueValue})
						if err != nil {
							return true, nil
						}
					}
				}
			}
		}
		return false, nil
	})
}

func CreateCredentialSecret(ctx context.Context, client kubernetes.Interface, cluster *api.Cluster, owner string) error {
	cred, err := Store(ctx).Owner(owner).Credentials().Get(cluster.Spec.Config.CredentialName)
	if err != nil {
		return err
	}
	secret := &core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: cluster.ClusterConfig().Cloud.CloudProvider,
		},
		StringData: cred.Spec.Data,
		Type:       core.SecretTypeOpaque,
	}

	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		_, err := client.CoreV1().Secrets(metav1.NamespaceSystem).Get(secret.Name, metav1.GetOptions{})
		if err == nil {
			return true, nil
		}
		_, err = client.CoreV1().Secrets(metav1.NamespaceSystem).Create(secret)
		if err != nil {
			return false, nil
		}
		return false, nil
	})
}

func NewClusterApiClient(ctx context.Context, cluster *api.Cluster) (client.Client, error) {
	adminCert, adminKey, err := CreateAdminCertificate(ctx)
	if err != nil {
		return nil, err
	}
	host := cluster.APIServerURL()
	if host == "" {
		return nil, errors.Errorf("failed to detect api server url for cluster %s", cluster.Name)
	}
	cfg := &rest.Config{
		Host: host,
		TLSClientConfig: rest.TLSClientConfig{
			CAData:   cert.EncodeCertPEM(CACert(ctx)),
			CertData: cert.EncodeCertPEM(adminCert),
			KeyData:  cert.EncodePrivateKeyPEM(adminKey),
		},
	}
	return client.New(cfg, client.Options{})
}

func waitForServiceAccount(ctx context.Context, client kubernetes.Interface) error {
	attempt := 0
	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		attempt++
		Logger(ctx).Infof("Attempt %v: Waiting for the service account to exist...", attempt)

		_, err := client.CoreV1().ServiceAccounts(ServiceAccountNs).Get(ServiceAccountName, metav1.GetOptions{})
		return err == nil, nil
	})
}

func CreateSecret(kc kubernetes.Interface, name, namespace string, data map[string][]byte) error {
	secret := &core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type: core.SecretTypeOpaque,
		Data: data,
	}
	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		if _, err := kc.CoreV1().Secrets(namespace).Get(secret.Name, metav1.GetOptions{}); err == nil {
			fmt.Println("Secret %q Already Exists, Ignoring", secret.Name)
			return true, nil
		}

		_, err := kc.CoreV1().Secrets(namespace).Create(secret)
		fmt.Println(err)
		return err == nil, nil
	})
}

func CreateNamespace(kc kubernetes.Interface, namespace string) error {
	ns := &core.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
			Labels: map[string]string{
				"controller-tools.k8s.io": "1.0",
			},
		},
	}
	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		if _, err := kc.CoreV1().Namespaces().Get(ns.Name, metav1.GetOptions{}); err == nil {
			fmt.Printf("Namespace %q Already Exists, Ignoring", ns.Name)
			return true, nil
		}

		_, err := kc.CoreV1().Namespaces().Create(ns)
		fmt.Println(err)
		return err == nil, nil
	})
}

func CreateConfigMap(kc kubernetes.Interface, name, namespace string, data map[string]string) error {
	conf := &core.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}
	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		_, err := kc.CoreV1().ConfigMaps(namespace).Create(conf)

		fmt.Println(err)
		return err == nil, nil
	})
}
