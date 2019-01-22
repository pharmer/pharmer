package describer

import (
	"fmt"
	"io"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/kubectl/describe"
	"k8s.io/kubernetes/pkg/printers"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

const statusUnknown = "Unknown"

func (d *humanReadableDescriber) describeCluster(item *api.Cluster, describerSettings describe.DescriberSettings) (string, error) {

	nodeGroups, err := cloud.Store(d.ctx).MachineSet(item.Name).List(metav1.ListOptions{})
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

func describeMachineSets(nodeGroups []*clusterv1.MachineSet, out io.Writer) {
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
