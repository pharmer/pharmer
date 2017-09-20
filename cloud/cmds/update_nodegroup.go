package cmds

import (
	"context"
	"fmt"

	"github.com/appscode/go-term"
	"github.com/appscode/go/flags"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/cloud/util"
	"github.com/appscode/pharmer/config"
	"github.com/spf13/cobra"
)

func NewCmdUpdateNodeGroup() *cobra.Command {
	cmd := &cobra.Command{
		Use: api.ResourceNameNodeGroup,
		Aliases: []string{
			api.ResourceTypeNodeGroup,
			api.ResourceCodeNodeGroup,
			api.ResourceKindNodeGroup,
		},
		Short:             "Update Kubernetes cluster NodeGroup",
		Example:           `pharmer update nodegroup -k <cluster_name>`,
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			flags.EnsureRequiredFlags(cmd, "file")

			if len(args) == 0 {
				term.Fatalln("Missing nodegroup name.")
			}
			if len(args) > 1 {
				term.Fatalln("Multiple nodegroup name provided.")
			}

			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				term.Fatalln(err)
			}
			ctx := cloud.NewContext(context.Background(), cfg)

			if err := runUpdateNodeGroup(ctx, cmd, args); err != nil {
				term.Fatalln(err)
			}
		},
	}

	cmd.Flags().StringP("cluster", "k", "", "Name of the Kubernetes cluster")
	cmd.Flags().StringP("file", "f", "", "Load nodegroup data from file")

	return cmd
}

func runUpdateNodeGroup(ctx context.Context, cmd *cobra.Command, args []string) error {
	clusterName, _ := cmd.Flags().GetString("cluster")
	nodeGroup, err := cloud.Store(ctx).NodeGroups(clusterName).Get(args[0])
	if err != nil {
		return fmt.Errorf(`NodeGroup "%v" not found.`, nodeGroup)
	}

	fileName, _ := cmd.Flags().GetString("file")

	var updatedNodeGroup *api.NodeGroup
	if err := util.ReadFileAs(fileName, &updatedNodeGroup); err != nil {
		return err
	}

	nodeGroup.Spec = updatedNodeGroup.Spec
	_, err = cloud.Store(ctx).NodeGroups(clusterName).Update(nodeGroup)
	return err
}
