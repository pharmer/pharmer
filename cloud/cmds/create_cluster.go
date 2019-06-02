package cmds

import (
	"github.com/appscode/go/term"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/cloud/cmds/options"
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
		Short:             "Create a Kubernetes cluster for a given cloud provider",
		Example:           "pharmer create cluster demo-cluster",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.ValidateFlags(cmd, args); err != nil {
				term.Fatalln(err)
			}

			err := store.SetProvider(cmd, opts.Owner)
			if err != nil {
				term.Fatalln(err)
			}
			err = runCreateClusterCmd(store.StoreProvider, opts)
			if err != nil {
				term.Fatalln(err)
			}

		},
	}
	opts.AddFlags(cmd.Flags())

	return cmd
}

func runCreateClusterCmd(store store.ResourceInterface, opts *options.ClusterCreateConfig) error {
	cm, err := cloud.Create(store, opts.Cluster)
	if err != nil {
		return err
	}

	if len(opts.Nodes) > 0 {
		nodeOpts := options.NewNodeGroupCreateConfig()
		nodeOpts.ClusterName = cm.GetCluster().Name
		nodeOpts.Nodes = opts.Nodes
		err := cloud.CreateMachineSets(store, cm, nodeOpts)
		if err != nil {
			return err
		}
	}

	return nil
}
