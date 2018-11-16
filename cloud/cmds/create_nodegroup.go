package cmds

import (
	"context"

	"github.com/appscode/go/term"
	api "github.com/pharmer/pharmer/apis/v1alpha1"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/cloud/cmds/options"
	"github.com/pharmer/pharmer/config"
	"github.com/spf13/cobra"
)

func NewCmdCreateNodeGroup() *cobra.Command {
	opts := options.NewNodeGroupCreateConfig()
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
			err := opts.ValidateFlags(cmd, args)
			if err != nil {
				term.Fatalln(err)
			}

			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			term.ExitOnError(err)

			ctx := cloud.NewContext(context.Background(), cfg, config.GetEnv(cmd.Flags()))

			CreateNodeGroups(ctx, opts)

		},
	}
	opts.AddFlags(cmd.Flags())

	return cmd
}

func CreateNodeGroups(ctx context.Context, opts *options.NodeGroupCreateConfig) {
	cluster, err := cloud.Get(ctx, opts.ClusterName, opts.Owner)
	term.ExitOnError(err)
	for sku, count := range opts.Nodes {
		err := cloud.CreateNodeGroup(ctx, cluster, opts.Owner, api.RoleNode, sku, api.NodeType(opts.NodeType), count, opts.SpotPriceMax)
		term.ExitOnError(err)
	}
}
