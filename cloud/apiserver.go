package cloud

import (
	"context"
	"fmt"
	"strings"
	"time"

	stringz "github.com/appscode/go/strings"
	api "github.com/pharmer/pharmer/apis/v1"
	apiAlpha "github.com/pharmer/pharmer/apis/v1alpha1"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/cert"
	clustercommon "sigs.k8s.io/cluster-api/pkg/apis/cluster/common"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset"
	client "sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset/typed/cluster/v1alpha1"
)

const (
	RetryInterval      = 5 * time.Second
	RetryTimeout       = 15 * time.Minute
	ServiceAccountNs   = "kube-system"
	ServiceAccountName = "default"
)

func NodeCount(nodeGroups []*apiAlpha.NodeGroup) int64 {
	count := int64(0)
	for _, ng := range nodeGroups {
		count += ng.Spec.Nodes
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

func FindMasterMachines(cluster *api.Cluster) ([]*clusterv1.Machine, error) {
	if len(cluster.Spec.Masters) == 0 {
		return nil, fmt.Errorf("master machine not found")
	}
	return cluster.Spec.Masters, nil
}

func RoleContains(a clustercommon.MachineRole, list []clustercommon.MachineRole) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func IsMaster(machine *clusterv1.Machine) bool {
	return RoleContains(clustercommon.MasterRole, machine.Spec.Roles)
}

func IsHASetup(cluster *api.Cluster) bool {
	masters, err := FindMasterMachines(cluster)
	if err != nil {
		return false
	}
	return len(masters) > 1
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
		fmt.Println(err, ",.,.,.,.,.,")
		return err == nil, nil
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

func CreateCredentialSecret(ctx context.Context, client kubernetes.Interface, cluster *api.Cluster) error {
	cred, err := Store(ctx).Credentials().Get(cluster.Spec.CredentialName)
	if err != nil {
		return err
	}
	secret := &core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: cluster.ProviderConfig().CloudProvider,
		},
		StringData: cred.Spec.Data,
		Type:       core.SecretTypeOpaque,
	}

	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		_, err := client.CoreV1().Secrets(metav1.NamespaceSystem).Create(secret)
		return err == nil, nil
	})
}

func NewClusterApiClient(ctx context.Context, cluster *api.Cluster) (*clientset.Clientset, error) {
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
	return clientset.NewForConfig(cfg)
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

func waitForClusterResourceReady(ctx context.Context, clientSet clientset.Interface) error {
	attempt := 0
	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		attempt++
		Logger(ctx).Infof("Attempt %v: Probing Kubernetes api server ...", attempt)
		_, err := clientSet.Discovery().ServerResourcesForGroupVersion("cluster.k8s.io/v1alpha1")
		fmt.Println(err)
		return err == nil, nil
	})
}

func GetCurrentMachineIfExists(machineClient client.MachineInterface, machine *clusterv1.Machine) (*clusterv1.Machine, error) {
	return GetMachineIfExists(machineClient, machine.ObjectMeta.Name, machine.ObjectMeta.UID)
}

func GetMachineIfExists(machineClient client.MachineInterface, name string, uid types.UID) (*clusterv1.Machine, error) {
	if machineClient == nil {
		fmt.Println("machine client is nil")
		// Being called before k8s is setup as part of master VM creation
		return nil, nil
	}

	// Machines are identified by name and UID
	machine, err := machineClient.Get(name, metav1.GetOptions{})
	if err != nil {
		// TODO: Use formal way to check for not found
		if strings.Contains(err.Error(), "not found") {
			return nil, nil
		}
		return nil, err
	}

	fmt.Println(name, "<><>", machine.ObjectMeta.UID, "<<<", uid)

	if machine.ObjectMeta.UID != uid {
		fmt.Println("uid not match")
		return nil, nil
	}
	return machine, nil
}

func CreateSecret(kc kubernetes.Interface, name string, data map[string][]byte) error {
	secret := &core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Data: data,
	}
	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		_, err := kc.CoreV1().Secrets(metav1.NamespaceDefault).Create(secret)
		fmt.Println(err)
		return err == nil, nil
	})
}
