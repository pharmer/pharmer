package cmds

import (
	"context"

	"github.com/appscode/go-term"
	"github.com/appscode/go/flags"
	api "github.com/appscode/pharmer/apis/v1alpha1"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/config"
	"github.com/spf13/cobra"
)

func NewCmdCreateNodeGroup() *cobra.Command {

	nodes := map[string]int{}

	cmd := &cobra.Command{
		Use: api.ResourceNameNodeGroup,
		Aliases: []string{
			api.ResourceTypeNodeGroup,
			api.ResourceCodeNodeGroup,
			api.ResourceKindNodeGroup,
		},
		Short:             "Create a Kubernetes cluster NodeGroup for a given cloud provider",
		Example:           "pharmer create nodegroup -k <cluster_name>",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			flags.EnsureRequiredFlags(cmd, "cluster", "nodes")

			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			term.ExitOnError(err)

			ctx := cloud.NewContext(context.Background(), cfg, config.GetEnv(cmd.Flags()))

			clusterName, _ := cmd.Flags().GetString("cluster")
			cluster, err := cloud.Get(ctx, clusterName)
			term.ExitOnError(err)
			CreateNodeGroups(ctx, cluster, nodes)

		},
	}

	cmd.Flags().StringP("cluster", "k", "", "Name of the Kubernetes cluster")
	cmd.Flags().StringToIntVar(&nodes, "nodes", map[string]int{}, "Node set configuration")

	return cmd
}

func CreateNodeGroups(ctx context.Context, cluster *api.Cluster, nodes map[string]int) {
	for sku, count := range nodes {
		err := cloud.CreateNodeGroup(ctx, cluster, api.RoleNode, sku, count)
		term.ExitOnError(err)
	}
}
