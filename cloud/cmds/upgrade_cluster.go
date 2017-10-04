package cmds

import (
	"context"
	"fmt"

	"github.com/appscode/go-term"
	"github.com/appscode/go/flags"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/config"
	"github.com/spf13/cobra"
)

func NewCmdUpgradeCluster() *cobra.Command {
	cmd := &cobra.Command{
		Use: api.ResourceNameCluster,
		Aliases: []string{
			api.ResourceTypeCluster,
			api.ResourceKindCluster,
		},
		Short:             "Upgrade cluster object",
		Example:           `pharmer upgrade cluster -k <cluster-name>  v1.8.0 `,
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			flags.EnsureRequiredFlags(cmd, "cluster")

			clusterName, _ := cmd.Flags().GetString("cluster")

			if len(args) == 0 {
				term.Fatalln("Missing cluster version.")
			}
			if len(args) > 1 {
				term.Fatalln("Multiple cluster version provided.")
			}

			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				term.Fatalln(err)
			}
			ctx := cloud.NewContext(context.Background(), cfg, config.GetEnv(cmd.Flags()))

			if err := runUpgradeCluster(ctx, clusterName, args); err != nil {
				term.Fatalln(err)
			}
		},
	}

	cmd.Flags().StringP("cluster", "k", "", "Name of the Kubernetes cluster")

	return cmd
}

func runUpgradeCluster(ctx context.Context, clusterName string, args []string) error {

	cluster, err := cloud.Get(ctx, clusterName)
	if err != nil {
		return fmt.Errorf(`Cluster "%v" not found.`, cluster)
	}

	cluster.Spec.KubernetesVersion = args[0]
	_, err = cloud.Upgrade(ctx, cluster)
	return err
}
