package cloud

import (
	"github.com/appscode/go/term"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/cmds/cloud/options"
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

			storeProvider, err := store.GetStoreProvider(cmd)
			if err != nil {
				term.Fatalln(err)
			}

			cluster, err := storeProvider.Clusters().Get(opts.ClusterName)
			if err != nil {
				term.Fatalln(err)
			}

			scope := cloud.NewScope(cloud.NewScopeParams{
				Cluster:       cluster,
				StoreProvider: storeProvider,
			})
			cm, err := scope.GetCloudManager()
			term.ExitOnError(err)

			kubeconfig, err := cm.GetKubeConfig()
			term.ExitOnError(err)

			err = cloud.UseCluster(opts, kubeconfig)
			if err != nil {
				term.Fatalln(err)
			}
		},
	}
	opts.AddFlags(cmd.Flags())
	return cmd
}
