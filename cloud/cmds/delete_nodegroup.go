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

func NewCmdDeleteNodeGroup() *cobra.Command {
	ngConfig := options.NewNodeGroupDeleteConfig()
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
			if err := ngConfig.ValidateNodeGroupDeleteFlags(cmd, args); err != nil {
				term.Fatalln(err)
			}
			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			term.ExitOnError(err)

			ctx := cloud.NewContext(context.Background(), cfg, config.GetEnv(cmd.Flags()))
			nodeGroups, err := getNodeGroupList(ctx, ngConfig.ClusterName, args...)
			term.ExitOnError(err)

			for _, ng := range nodeGroups {
				err := cloud.DeleteNG(ctx, ng.Name, ngConfig.ClusterName)
				term.ExitOnError(err)
			}
		},
	}
	ngConfig.AddNodeGroupDeleteFlags(cmd.Flags())

	return cmd
}
