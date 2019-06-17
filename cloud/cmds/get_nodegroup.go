package cmds

import (
	"io"

	"github.com/appscode/go/term"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud/cmds/options"
	"github.com/pharmer/pharmer/store"
	"github.com/pharmer/pharmer/utils/printer"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

			err := RunGetNodeGroup(opts, out)
			term.ExitOnError(err)
		},
	}
	opts.AddFlags(cmd.Flags())

	return cmd
}

func RunGetNodeGroup(opts *options.NodeGroupGetConfig, out io.Writer) error {
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
		clusters, err := store.StoreProvider.Clusters().List(metav1.ListOptions{})
		if err != nil {
			return err
		}
		for _, c := range clusters {
			clusterList = append(clusterList, c.Name)
		}
	}

	for _, cluster := range clusterList {
		nodegroups, err := GetMachineSetList(cluster, opts.NodeGroups...)
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
			printer.PrintNewline(w)
		}

	}

	w.Flush()
	return nil
}

func GetMachineSetList(cluster string, args ...string) ([]*clusterv1.MachineSet, error) {
	var machineSetList []*clusterv1.MachineSet
	if len(args) != 0 {
		for _, arg := range args {
			ms, err := store.StoreProvider.MachineSet(cluster).Get(arg)
			if err != nil {
				return nil, err
			}
			machineSetList = append(machineSetList, ms)
		}
	} else {
		var err error
		machineSetList, err = store.StoreProvider.MachineSet(cluster).List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
	}
	return machineSetList, nil
}

func GetNodeGroupList(cluster string, args ...string) (nodeGroupList []*api.NodeGroup, err error) {
	if len(args) != 0 {
		for _, arg := range args {
			nodeGroup, er2 := store.StoreProvider.NodeGroups(cluster).Get(arg)
			if er2 != nil {
				return nil, er2
			}
			nodeGroupList = append(nodeGroupList, nodeGroup)
		}

	} else {
		nodeGroupList, err = store.StoreProvider.NodeGroups(cluster).List(metav1.ListOptions{})
		if err != nil {
			return
		}
	}
	return
}
