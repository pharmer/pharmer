package cmds

import (
	"context"

	"github.com/appscode/go/flags"
	"github.com/appscode/go/term"
	api "github.com/appscode/pharmer/apis/v1alpha1"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/config"
	"github.com/spf13/cobra"
)

func NewCmdCreateNodeGroup() *cobra.Command {
	var spotInstance bool
	var spotPriceMax float64
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
			ensureFlags := []string{"cluster", "nodes"}
			if spotInstance {
				ensureFlags = append(ensureFlags, "spot-price-max")
			}
			flags.EnsureRequiredFlags(cmd, ensureFlags...)

			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			term.ExitOnError(err)

			ctx := cloud.NewContext(context.Background(), cfg, config.GetEnv(cmd.Flags()))

			clusterName, _ := cmd.Flags().GetString("cluster")
			cluster, err := cloud.Get(ctx, clusterName)
			term.ExitOnError(err)
			CreateNodeGroups(ctx, cluster, nodes, spotInstance, spotPriceMax)

		},
	}

	cmd.Flags().StringP("cluster", "k", "", "Name of the Kubernetes cluster")
	cmd.Flags().BoolVar(&spotInstance, "spot-instance", false, "Set spot instance flag")
	cmd.Flags().Float64Var(&spotPriceMax, "spot-price-max", float64(0), "Maximum price of spot instance")
	cmd.Flags().StringToIntVar(&nodes, "nodes", map[string]int{}, "Node set configuration")

	return cmd
}

func CreateNodeGroups(ctx context.Context, cluster *api.Cluster, nodes map[string]int, spotInstance bool, spotPriceMax float64) {
	for sku, count := range nodes {
		err := cloud.CreateNodeGroup(ctx, cluster, api.RoleNode, sku, count, spotInstance, spotPriceMax)
		term.ExitOnError(err)
	}
}
