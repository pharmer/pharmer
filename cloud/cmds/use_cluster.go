package cmds

import (
	"context"

	"github.com/appscode/go/log"
	"github.com/appscode/go/term"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/cloud/cmds/options"
	"github.com/pharmer/pharmer/config"
	"github.com/spf13/cobra"
)

func NewCmdUse() *cobra.Command {
	opts := options.NewClusterUseConfig()
	cmd := &cobra.Command{
		Use:               "cluster",
		Short:             "Sets `kubectl` context to given cluster",
		Example:           `pharmer use cluster <name>`,
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.ValidateFlags(cmd, args); err != nil {
				term.Fatalln(err)
			}
			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				log.Fatalln(err)
			}
			ctx := cloud.NewContext(context.Background(), cfg, config.GetEnv(cmd.Flags()))

			cluster, err := cloud.Store(ctx).Owner(opts.Owner).Clusters().Get(opts.ClusterName)
			if err != nil {
				term.Fatalln(err)
			}
			c2, err := cloud.GetAdminConfig(ctx, cluster, opts.Owner)
			if err != nil {
				log.Fatalln(err)
			}
			cloud.UseCluster(ctx, opts, c2)
		},
	}
	opts.AddFlags(cmd.Flags())
	return cmd
}
