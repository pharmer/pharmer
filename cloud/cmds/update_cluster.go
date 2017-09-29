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

func NewCmdUpdateCluster() *cobra.Command {
	cmd := &cobra.Command{
		Use: api.ResourceNameCluster,
		Aliases: []string{
			api.ResourceTypeCluster,
			api.ResourceKindCluster,
		},
		Short:             "Update cluster object",
		Example:           `pharmer update cluster`,
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			flags.EnsureRequiredFlags(cmd, "file")

			if len(args) == 0 {
				term.Fatalln("Missing cluster name.")
			}
			if len(args) > 1 {
				term.Fatalln("Multiple cluster name provided.")
			}

			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				term.Fatalln(err)
			}
			ctx := cloud.NewContext(context.Background(), cfg, config.GetEnv(cmd.Flags()))

			if err := runUpdateCluster(ctx, cmd, args); err != nil {
				term.Fatalln(err)
			}
		},
	}

	cmd.Flags().StringP("file", "f", "", "Load cluster data from file")

	return cmd
}

func runUpdateCluster(ctx context.Context, cmd *cobra.Command, args []string) error {

	cluster, err := cloud.Get(ctx, args[0])
	if err != nil {
		return fmt.Errorf(`Cluster "%v" not found.`, cluster)
	}

	fileName, _ := cmd.Flags().GetString("file")

	var updatedCluster *api.Cluster
	if err := util.ReadFileAs(fileName, &updatedCluster); err != nil {
		return err
	}

	cluster.Spec = updatedCluster.Spec
	_, err = cloud.Update(ctx, cluster)
	return err
}
