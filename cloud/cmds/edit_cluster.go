package cmds

import (
	"context"

	"github.com/appscode/go-term"
	"github.com/appscode/go/flags"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/config"
	"github.com/spf13/cobra"
)

func NewCmdEditCluster() *cobra.Command {
	var (
		KubernetesVersion string
	)
	cmd := &cobra.Command{
		Use: api.ResourceNameCluster,
		Aliases: []string{
			api.ResourceTypeCluster,
			api.ResourceKindCluster,
		},
		Short:             "Edit cluster object",
		Example:           `pharmer edit cluster <cluster-name> --kubernetes-version=v1.8.0 `,
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			flags.EnsureRequiredFlags(cmd, "kubernetes-version")

			if len(args) == 0 {
				term.Fatalln("Missing cluster name")
			}
			if len(args) > 1 {
				term.Fatalln("Multiple cluster name provided.")
			}
			clusterName := args[0]

			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				term.Fatalln(err)
			}
			ctx := cloud.NewContext(context.Background(), cfg, config.GetEnv(cmd.Flags()))

			if _, err := cloud.Edit(ctx, clusterName, KubernetesVersion); err != nil {
				term.Fatalln(err)
			}
		},
	}

	cmd.Flags().StringVar(&KubernetesVersion, "kubernetes-version", "", "Kubernetes version")
	return cmd
}
