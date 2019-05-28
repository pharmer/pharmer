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

			store.SetProvider(cmd, opts.Owner)

			cm, cluster, err := cloud.Create(opts.Cluster)
			if err != nil {
				term.Fatalln(err)
			}
			if len(opts.Nodes) > 0 {
				nodeOpts := options.NewNodeGroupCreateConfig()
				nodeOpts.ClusterName = cluster.Name
				nodeOpts.Nodes = opts.Nodes
				err := cloud.CreateMachineSetsFromOptions(cm, nodeOpts)
				if err != nil {
					term.Fatalln(err)
				}
			}
		},
	}
	opts.AddFlags(cmd.Flags())

	return cmd
}
