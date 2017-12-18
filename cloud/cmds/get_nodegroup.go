package cmds

import (
	"context"
	"io"

	"github.com/appscode/go/term"
	api "github.com/pharmer/pharmer/apis/v1alpha1"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/cloud/cmds/options"
	"github.com/pharmer/pharmer/config"
	"github.com/pharmer/pharmer/utils/printer"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			term.ExitOnError(err)

			ctx := cloud.NewContext(context.Background(), cfg, config.GetEnv(cmd.Flags()))
			RunGetNodeGroup(ctx, opts, out)

		},
	}
	opts.AddFlags(cmd.Flags())

	return cmd
}

func RunGetNodeGroup(ctx context.Context, opts *options.NodeGroupGetConfig, out io.Writer) error {

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
		clusters, err := cloud.Store(ctx).Clusters().List(metav1.ListOptions{})
		if err != nil {
			return err
		}
		for _, c := range clusters {
			clusterList = append(clusterList, c.Name)
		}
	}

	for _, cluster := range clusterList {
		nodegroups, err := GetNodeGroupList(ctx, cluster, opts.NodeGroups...)
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
			if rPrinter.IsGeneric() {
				printer.PrintNewline(w)
			}
		}

	}

	w.Flush()
	return nil
}

func GetNodeGroupList(ctx context.Context, cluster string, args ...string) (nodeGroupList []*api.NodeGroup, err error) {
	if len(args) != 0 {
		for _, arg := range args {
			nodeGroup, er2 := cloud.Store(ctx).NodeGroups(cluster).Get(arg)
			if er2 != nil {
				return nil, er2
			}
			nodeGroupList = append(nodeGroupList, nodeGroup)
		}

	} else {
		nodeGroupList, err = cloud.Store(ctx).NodeGroups(cluster).List(metav1.ListOptions{})
		if err != nil {
			return
		}
	}
	return
}
