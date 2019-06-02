package cmds

import (
	"github.com/appscode/go/log"
	"github.com/appscode/go/term"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/cloud/cmds/options"
	"github.com/pharmer/pharmer/store"
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

			err := store.SetProvider(cmd, opts.Owner)
			if err != nil {
				term.Fatalln(err)
			}

			cluster, err := store.StoreProvider.Clusters().Get(opts.ClusterName)
			if err != nil {
				term.Fatalln(err)
			}

			cm, err := cloud.GetCloudManager(cluster)
			if err != nil {
				term.Fatalln(err)
			}

			kubeconfig, err := cloud.GetAdminConfig(cm, cluster)
			if err != nil {
				log.Fatalln(err)
			}
			err = cloud.UseCluster(opts, kubeconfig)
			if err != nil {
				term.Fatalln(err)
			}
		},
	}
	opts.AddFlags(cmd.Flags())
	return cmd
}
