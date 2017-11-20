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

func NewCmdDeleteNodeGroup() *cobra.Command {
	cmd := &cobra.Command{
		Use: api.ResourceNameNodeGroup,
		Aliases: []string{
			api.ResourceTypeNodeGroup,
			api.ResourceCodeNodeGroup,
			api.ResourceKindNodeGroup,
		},
		Short:             "Delete a Kubernetes cluster NodeGroup",
		Example:           "pharmer delete nodegroup -k <cluster_name>",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			flags.EnsureRequiredFlags(cmd, "cluster")

			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			term.ExitOnError(err)

			ctx := cloud.NewContext(context.Background(), cfg, config.GetEnv(cmd.Flags()))
			clusterName, _ := cmd.Flags().GetString("cluster")

			nodeGroups, err := getNodeGroupList(ctx, clusterName, args...)
			term.ExitOnError(err)

			for _, ng := range nodeGroups {
				err := cloud.DeleteNG(ctx, ng.Name, clusterName)
				term.ExitOnError(err)
			}
		},
	}

	cmd.Flags().StringP("cluster", "k", "", "Name of the Kubernetes cluster")
	return cmd
}
