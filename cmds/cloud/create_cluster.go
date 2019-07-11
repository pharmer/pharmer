package cloud

import (
	"github.com/appscode/go/term"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/cmds/cloud/options"
	"github.com/pharmer/pharmer/store"
	"github.com/spf13/cobra"
	"k8s.io/klog/klogr"
)

func NewCmdCreateCluster() *cobra.Command {
	opts := options.NewClusterCreateConfig()
	cmd := &cobra.Command{
		Use: api.ResourceNameCluster,
		Aliases: []string{
			api.ResourceTypeCluster,
			api.ResourceKindCluster,
		},
		Short:             "Create a Kubernetes cluster for a given cloud provider",
		Example:           "pharmer create cluster demo-cluster",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.ValidateFlags(cmd, args); err != nil {
				term.Fatalln(err)
			}

			storeProvider, err := store.GetStoreProvider(cmd)
			if err != nil {
				term.Fatalln(err)
			}

			err = runCreateClusterCmd(storeProvider, opts)
			if err != nil {
				term.Fatalln(err)
			}
		},
	}
	opts.AddFlags(cmd.Flags())

	return cmd
}

func runCreateClusterCmd(store store.ResourceInterface, opts *options.ClusterCreateConfig) error {
	scope := cloud.NewScope(cloud.NewScopeParams{
		Cluster:       opts.Cluster,
		StoreProvider: store,
		Logger:        klogr.New().WithValues("cluster-name", opts.Cluster.Name),
	})

	err := cloud.CreateCluster(scope)
	if err != nil {
		return err
	}

	if len(opts.Nodes) > 0 {
		nodeOpts := options.NewNodeGroupCreateConfig()
		nodeOpts.ClusterName = opts.Cluster.Name
		nodeOpts.Nodes = opts.Nodes
		err := cloud.CreateMachineSets(store, nodeOpts)
		if err != nil {
			return err
		}
	}

	return nil
}
