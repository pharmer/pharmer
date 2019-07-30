package describer

import (
	"fmt"
	"io"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/kubectl/describe"
	"k8s.io/kubernetes/pkg/printers"
	api "pharmer.dev/pharmer/apis/v1alpha1"
	"pharmer.dev/pharmer/store"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (d *humanReadableDescriber) describeCluster(machinesetStore store.MachineSetStore, item *api.Cluster, describerSettings describe.DescriberSettings) (string, error) {
	nodeGroups, err := machinesetStore.List(metav1.ListOptions{})
	if err != nil {
		return "", err
	}

	return tabbedString(func(out io.Writer) error {
		fmt.Fprintf(out, "Name:\t%s\n", item.Name)
		fmt.Fprintf(out, "Version:\t%s\n", item.ClusterConfig().KubernetesVersion)
		describeMachineSets(nodeGroups, out)
		return nil
	})
}

func describeMachineSets(nodeGroups []*clusterapi.MachineSet, out io.Writer) {
	if len(nodeGroups) == 0 {
		fmt.Fprint(out, "No NodeGroup.\n")
		return
	}

	fmt.Fprint(out, "NodeGroup:\n")

	w := printers.GetNewTabWriter(out)

	fmt.Fprint(w, "  Name\tNode\n")
	fmt.Fprint(w, "  ----\t------\n")

	for _, ng := range nodeGroups {
		fmt.Fprintf(w, "  %s\t%v\n",
			ng.Name,
			*ng.Spec.Replicas,
		)
	}
	w.Flush()
}
