package cloud

import (
	"github.com/appscode/go/term"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/cmds/cloud/options"
	"github.com/pharmer/pharmer/store"
	"github.com/spf13/cobra"
)

func NewCmdCreateCluster() *cobra.Command {
	opts := options.NewClusterCreateConfig()
	cmd := &cobra.Command{
		Use: api.ResourceNameCluster,
		Aliases: []string{
			api.ResourceTypeCluster,
			api.ResourceKindCluster,
		},
		Short:             "CreateCluster a Kubernetes cluster for a given cloud provider",
		Example:           "pharmer create cluster demo-cluster",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.ValidateFlags(cmd, args); err != nil {
				term.Fatalln(err)
			}

			storeProvider, err := store.GetStoreProvider(cmd, opts.Owner)
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
	err := cloud.CreateCluster(store, opts.Cluster)
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
