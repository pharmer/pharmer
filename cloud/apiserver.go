package cloud

import (
	"context"
	"time"

	semver "github.com/appscode/go-version"
	stringz "github.com/appscode/go/strings"
	api "github.com/pharmer/pharmer/apis/v1alpha1"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/cert"
)

const (
	RetryInterval = 5 * time.Second
	RetryTimeout  = 15 * time.Minute
)

func NodeCount(nodeGroups []*api.NodeGroup) int64 {
	count := int64(0)
	for _, ng := range nodeGroups {
		count += ng.Spec.Nodes
	}
	return count
}

func FindMasterNodeGroup(nodeGroups []*api.NodeGroup) (*api.NodeGroup, error) {
	for _, ng := range nodeGroups {
		if ng.IsMaster() {
			return ng, nil
		}
	}
	return nil, ErrNoMasterNG
}

// WARNING:
// Returned KubeClient uses admin client cert. This should only be used for cluster provisioning operations.
func NewAdminClient(ctx context.Context, cluster *api.Cluster) (kubernetes.Interface, error) {
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
	cred, err := Store(ctx).Owner(owner).Credentials().Get(cluster.Spec.CredentialName)
	if err != nil {
		return err
	}
	secret := &core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: cluster.Spec.Cloud.CloudProvider,
		},
		StringData: cred.Spec.Data,
		Type:       core.SecretTypeOpaque,
	}

	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		_, err := client.CoreV1().Secrets(metav1.NamespaceSystem).Create(secret)
		return err == nil, nil
	})
}
