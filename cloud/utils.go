package cloud

import (
	"github.com/appscode/errors"
	stringutil "github.com/appscode/go/strings"
	"github.com/appscode/pharmer/contexts"
	"github.com/appscode/pharmer/storage"
	"github.com/cenkalti/backoff"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

var restrictedNamespaces []string = []string{"appscode", "kube-system"}

func hasNoUserApps(ctx context.Context, clusterName string) (bool, error) {
	context, err := contexts.NewKubeAddonContext(ctx, clusterName)
	if err != nil {
		// If we can't connect to kube apiserver, then delete cluster.
		// Cluster probably failed to create.
		return true, nil
	}
	// TODO(tamal): Explore what is a good option here?
	if context.Status == storage.KubernetesStatus_Deleted ||
		// context.Status == storage.KubernetesStatus_Deleting ||
		// context.Status == storage.KubernetesStatus_Pending ||
		// context.Status == storage.KubernetesStatus_Failing ||
		context.Status == storage.KubernetesStatus_Failed {
		return false, errors.New().WithMessagef("Cluster %v is already %v.", context.Name, context.Status).Err()
	}

	pods := &apiv1.PodList{}
	pods, err = context.Client.CoreV1().Pods(apiv1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		// If we can't connect to kube apiserver, then delete cluster.
		// Cluster probably failed to create.
		return true, nil
	}
	for _, pod := range pods.Items {
		if pod.Status.Phase == "Running" && !stringutil.Contains(restrictedNamespaces, pod.Namespace) {
			return false, nil
		}
	}
	return true, nil
}

func deleteLoadBalancers(ctx *contexts.ClusterContext) error {
	context, err := ctx.NewKubeClient()
	if err != nil {
		return errors.New().Err()
	}

	// Delete ingresses
	backoff.Retry(func() error {
		ingresses, err := context.Client.Extensions().Ingresses("").List(metav1.ListOptions{})
		if err != nil {
			return err
		}
		for _, ingress := range ingresses.Items {
			trueValue := true
			err = context.Client.Extensions().Ingresses(ingress.Namespace).Delete(ingress.Name, &metav1.DeleteOptions{OrphanDependents: &trueValue})
			if err != nil {
				return err
			}
		}

		engresses, err := context.VoyagerClient.Ingresses("").List(metav1.ListOptions{})
		if err != nil {
			return err
		}
		for _, engress := range engresses.Items {
			err = context.VoyagerClient.Ingresses(engress.Namespace).Delete(engress.Name)
			if err != nil {
				return err
			}
		}

		return nil
	}, backoff.NewExponentialBackOff())

	// Delete services with type = LoadBalancer
	backoff.Retry(func() error {
		svcs, err := context.Client.CoreV1().Services("").List(metav1.ListOptions{})
		if err != nil {
			return err
		}
		for _, svc := range svcs.Items {
			if svc.Spec.Type == apiv1.ServiceTypeLoadBalancer {
				trueValue := true
				err = context.Client.CoreV1().Services(svc.Namespace).Delete(svc.Name, &metav1.DeleteOptions{OrphanDependents: &trueValue})
				if err != nil {
					return err
				}
			}
		}
		return nil
	}, backoff.NewExponentialBackOff())

	return nil
}

func deleteDyanamicVolumes(ctx *contexts.ClusterContext) error {
	context, err := ctx.NewKubeClient()
	if err != nil {
		return errors.New().Err()
	}

	backoff.Retry(func() error {
		pvcs, err := context.Client.CoreV1().PersistentVolumeClaims("").List(metav1.ListOptions{})
		if err != nil {
			return err
		}
		for _, pvc := range pvcs.Items {
			if pvc.Status.Phase == apiv1.ClaimBound {
				for k, v := range pvc.Annotations {
					if (k == "volume.alpha.kubernetes.io/storage-class" ||
						k == "volume.beta.kubernetes.io/storage-class") && v != "" {
						trueValue := true
						err = context.Client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Delete(pvc.Name, &metav1.DeleteOptions{OrphanDependents: &trueValue})
						if err != nil {
							return err
						}
					}
				}
			}
		}
		return nil
	}, backoff.NewExponentialBackOff())
	return nil
}
