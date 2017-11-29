package cmds

import (
	"context"
	"fmt"

	"github.com/appscode/go/flags"
	"github.com/appscode/go/term"
	api "github.com/appscode/pharmer/apis/v1alpha1"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/config"
	"github.com/spf13/cobra"
)

func NewCmdCreateNodeGroup() *cobra.Command {
	var nodeType string
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
			if api.NodeType(nodeType) == api.NodeTypeSpot {
				ensureFlags = append(ensureFlags, "spot-price-max")
			}
			flags.EnsureRequiredFlags(cmd, ensureFlags...)

			switch api.NodeType(nodeType) {
			case api.NodeTypeSpot, api.NodeTypeRegular:
				break
			default:
				term.Fatalln(fmt.Sprintf("flag [type] must be %v or %v", api.NodeTypeRegular, api.NodeTypeSpot))

			}

			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			term.ExitOnError(err)

			ctx := cloud.NewContext(context.Background(), cfg, config.GetEnv(cmd.Flags()))

			clusterName, _ := cmd.Flags().GetString("cluster")
			cluster, err := cloud.Get(ctx, clusterName)
			term.ExitOnError(err)
			CreateNodeGroups(ctx, cluster, nodes, api.NodeType(nodeType), spotPriceMax)

		},
	}

	cmd.Flags().StringP("cluster", "k", "", "Name of the Kubernetes cluster")
	cmd.Flags().StringVar(&nodeType, "type", string(api.NodeTypeRegular), "Set node type regular/spot, default regular")
	cmd.Flags().Float64Var(&spotPriceMax, "spot-price-max", float64(0), "Maximum price of spot instance")
	cmd.Flags().StringToIntVar(&nodes, "nodes", map[string]int{}, "Node set configuration")

	return cmd
}

func CreateNodeGroups(ctx context.Context, cluster *api.Cluster, nodes map[string]int, nodeType api.NodeType, spotPriceMax float64) {
	for sku, count := range nodes {
		err := cloud.CreateNodeGroup(ctx, cluster, api.RoleNode, sku, nodeType, count, spotPriceMax)
		term.ExitOnError(err)
	}
}
