package cloud

import (
	"context"
	"fmt"

	"github.com/appscode/go/log"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type InstanceType struct {
	ContextVersion int64
	Sku            string
	SpotInstance   bool
	Master         bool
	DiskType       string
	DiskSize       int64
}

func (t *InstanceType) String() string {
	return fmt.Sprintf("C:%0v, S:%v, P:%v, M:%v", t.ContextVersion, t.Sku, t.SpotInstance, t.Master)
}

type GroupStats struct {
	Count int64
	Extra interface{}
}

type Instance struct {
	Type  InstanceType
	Stats GroupStats
}
type InstanceController struct {
	Client kubernetes.Interface
}

func Mutator(ctx context.Context, cluster *api.Cluster, expectedInstance Instance, nodeGroup string) (int64, error) {
	return 0, nil
}

func getSKUFromNG(cluster, ng string) string {
	return ng[len(cluster)+1:]
}

//Deprecated
func GetClusterIstance(ctx context.Context, cluster *api.Cluster, nodeGroup string) ([]string, error) {
	var kc kubernetes.Interface // TODO: Fix NPE, pass client
	nodes, err := kc.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	existingNodes := make([]string, 0)
	for _, node := range nodes.Items {
		nl := api.FromMap(node.GetLabels())
		if nl.GetString(api.NodePoolKey) != nodeGroup {
			continue
		}
		existingNodes = append(existingNodes, node.Name)
	}
	return existingNodes, nil

}

func GetClusterIstance2(kc kubernetes.Interface, nodeGroup string) ([]string, error) {
	nodes, err := kc.CoreV1().Nodes().List(metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			api.NodePoolKey: nodeGroup,
		}).String(),
	})
	if err != nil {
		return nil, err
	}
	existingNodes := make([]string, 0)
	for _, node := range nodes.Items {
		nl := api.FromMap(node.GetLabels())
		if nl.GetString(api.NodePoolKey) != nodeGroup {
			continue
		}
		existingNodes = append(existingNodes, node.Name)
	}
	return existingNodes, nil

}

//Deprecated
func DeleteClusterInstance(ctx context.Context, cluster *api.Cluster, node string) error {
	var kc kubernetes.Interface // TODO: Fix NPE, pass client
	err := kc.CoreV1().Nodes().Delete(node, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	return nil
}

func DeleteClusterInstance2(kc kubernetes.Interface, node string) error {
	return kc.CoreV1().Nodes().Delete(node, &metav1.DeleteOptions{})
}

func GetExistingContextVersion(ctx context.Context, cluster *api.Cluster, sku string) (int64, error) {
	var kc kubernetes.Interface // TODO: Fix NPE, pass client
	//re, _ := labels.NewRequirement(api.NodeLabelKey_SKU, selection.Equals, []string{sku})
	nodes, err := kc.CoreV1().Nodes().List(metav1.ListOptions{
		//LabelSelector: labels.Selector.Add(*re).Matches(labels.Labels(api.NodeLabelKey_SKU)),
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, n := range nodes.Items {
		nl := api.FromMap(n.GetLabels())
		if nl.GetString(api.NodeLabelKey_SKU) == sku {
			return nl.GetInt64(api.NodeLabelKey_ContextVersion), nil
		}
	}
	return int64(0), errors.New("Context version not found")
}
