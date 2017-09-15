package cloud

import (
	"context"
	"fmt"
	"log"

	"github.com/appscode/go/errors"
	"github.com/appscode/pharmer/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
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
	Client clientset.Interface
}

func Mutator(ctx context.Context, cluster *api.Cluster, expectedInstance Instance, nodeGroup string) (int64, error) {
	kc, err := NewAdminClient(ctx, cluster)
	nodes, err := kc.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return expectedInstance.Stats.Count, err
	}

	desiredNGs := make(map[InstanceType]GroupStats)
	existingNGs := make(map[InstanceType]GroupStats)

	for _, n := range nodes.Items {
		nl := api.FromMap(n.GetLabels())
		if nl.GetString(api.NodeLabelKey_NodeGroup) != nodeGroup {
			continue
		}
		k := InstanceType{
			Sku:          getSKUFromNG(cluster.Name, nl.GetString(api.NodeLabelKey_NodeGroup)),
			SpotInstance: false,
			Master:       nl.GetString(api.NodeLabelKey_Role) == "master",
		}
		if gs, found := existingNGs[k]; !found {
			existingNGs[k] = GroupStats{
				Count: 1,
			}
		} else {
			gs.Count = gs.Count + 1
			existingNGs[k] = gs
		}
	}

	// compute diff
	diffNGs := make(map[InstanceType]GroupStats)

	if eGS, found := existingNGs[expectedInstance.Type]; found {
		if expectedInstance.Stats.Count != eGS.Count {
			diffNGs[expectedInstance.Type] = GroupStats{
				Count: expectedInstance.Stats.Count - eGS.Count,
				Extra: eGS.Extra,
			}
		}
	} else {
		diffNGs[expectedInstance.Type] = GroupStats{
			Count: expectedInstance.Stats.Count,
		}
	}

	//igm.DesiredInstance.Type = expectedInstance.Type
	//igm.DesiredInstance.Stats.Count =
	/*for k, eGS := range existingNGs {
		if _, found := desiredNGs[k]; !found {
			diffNGs[k] = GroupStats{
				Count: eGS.Count,
				Extra: eGS.Extra,
			}
		}
	}*/

	fmt.Println("existingNGs")
	for k, v := range existingNGs {
		fmt.Println(k.String(), " = ", v.Count)
	}

	fmt.Println("desiredNGs")
	for k, v := range desiredNGs {
		fmt.Println(k.String(), " = ", v.Count)
	}

	fmt.Println("diffNGs")
	for k, v := range diffNGs {
		fmt.Println(k.String(), " = ", v.Count)
	}

	// add nodes
	//var additions, deletions int64
	//var addGroups, delGroups int64
	var adjust int64
	adjust = diffNGs[expectedInstance.Type].Count

	/*for k := range ctx.NodeGroups {
		for x, y := range diffNGs {
			if ctx.NodeGroups[k].Sku == x.Sku {
				ctx.NodeGroups[k].Count += y.Count
				fmt.Println(ctx.NodeGroups[k].Count, "*********************************>>")
			}
			//ctx.NumNodes += v.Count
			//fmt.Println(k.String(), " = ", v.Count)
		}

	}*/
	/*fmt.Println(additions, "___--", addGroups, "---", additions+deletions)
	addCh := make(chan string, additions)
	delCh := make(chan string, deletions)

	for i := int64(0); i < additions+deletions; i++ {
		select {
		case a := <-addCh:
			fmt.Println("Added ", a)
		case d := <-delCh:
			fmt.Println("Deleted ", d)
		default:
			fmt.Println("default")
		}
	}*/

	// delete nodes
	return adjust, nil

}

func getSKUFromNG(cluster, ng string) string {
	return ng[len(cluster)+1:]
}

func GetClusterIstance(ctx context.Context, cluster *api.Cluster, nodeGroup string) ([]string, error) {
	kc, err := NewAdminClient(ctx, cluster)
	nodes, err := kc.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	existingNodes := make([]string, 0)
	for _, node := range nodes.Items {
		nl := api.FromMap(node.GetLabels())
		if nl.GetString(api.NodeLabelKey_NodeGroup) != nodeGroup {
			continue
		}
		existingNodes = append(existingNodes, node.Name)
	}
	return existingNodes, nil

}

func DeleteClusterInstance(ctx context.Context, cluster *api.Cluster, node string) error {
	kc, err := NewAdminClient(ctx, cluster)
	err = kc.CoreV1().Nodes().Delete(node, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	return nil
}

func GetExistingContextVersion(ctx context.Context, cluster *api.Cluster, sku string) (int64, error) {
	kc, err := NewAdminClient(ctx, cluster)
	if err != nil {
		log.Fatal(err)
	}
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
	return int64(0), errors.New("Context version not found").Err()
}

type Adder struct {
	ContextVersion int64
	Sku            string
	SpotInstance   bool
	Master         bool

	Count int64
	Extra interface{}
}

type Miner struct {
	ContextVersion int64
	Sku            string
	SpotInstance   bool
	Master         bool

	Count int64
	Extra interface{}
}
