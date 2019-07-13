package cloud

import (
	"io"

	"github.com/appscode/go/term"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	api "pharmer.dev/pharmer/apis/v1beta1"
	"pharmer.dev/pharmer/cmds/cloud/options"
	"pharmer.dev/pharmer/store"
	"pharmer.dev/pharmer/utils/printer"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func NewCmdGetNodeGroup(out io.Writer) *cobra.Command {
	opts := options.NewNodeGroupGetConfig()
	cmd := &cobra.Command{
		Use: api.ResourceNameNodeGroup,
		Aliases: []string{
			api.ResourceTypeNodeGroup,
			api.ResourceCodeNodeGroup,
			api.ResourceKindNodeGroup,
		},
		Short:             "Get a Kubernetes cluster NodeGroup",
		Example:           "pharmer get nodegroup -k <cluster_name>",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.ValidateFlags(cmd, args); err != nil {
				term.Fatalln(err)
			}

			storeProvider, err := store.GetStoreProvider(cmd)
			if err != nil {
				term.Fatalln(err)
			}

			err = runGetNodeGroup(storeProvider, opts, out)
			term.ExitOnError(err)
		},
	}
	opts.AddFlags(cmd.Flags())

	return cmd
}

func runGetNodeGroup(storeProvider store.ResourceInterface, opts *options.NodeGroupGetConfig, out io.Writer) error {
	rPrinter, err := printer.NewPrinter(opts.Output)
	if err != nil {
		return err
	}

	w := printer.GetNewTabWriter(out)

	clusterList := make([]string, 0)
	clusterName := opts.ClusterName

	if clusterName != "" {
		clusterList = append(clusterList, clusterName)
	} else {
		clusters, err := storeProvider.Clusters().List(metav1.ListOptions{})
		if err != nil {
			return err
		}
		for _, c := range clusters {
			clusterList = append(clusterList, c.Name)
		}
	}

	for _, cluster := range clusterList {
		nodegroups, err := getMachineSetList(storeProvider.MachineSet(cluster), opts.NodeGroups...)
		if err != nil {
			return err
		}
		if len(nodegroups) == 0 {
			continue
		}

		for _, ng := range nodegroups {
			if err := rPrinter.PrintObj(ng, w); err != nil {
				return err
			}
			err = printer.PrintNewline(w)
			if err != nil {
				return err
			}
		}

	}

	return w.Flush()
}

// TODO: move?
func getMachineSetList(machinesetStore store.MachineSetStore, args ...string) ([]*clusterv1.MachineSet, error) {
	var machineSetList []*clusterv1.MachineSet
	if len(args) != 0 {
		for _, arg := range args {
			ms, err := machinesetStore.Get(arg)
			if err != nil {
				return nil, err
			}
			machineSetList = append(machineSetList, ms)
		}
	} else {
		var err error
		machineSetList, err = machinesetStore.List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
	}
	return machineSetList, nil
}
