package cmds

import (
	"github.com/appscode/go/term"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/cloud/cmds/options"
	"github.com/pharmer/pharmer/store"
	"github.com/spf13/cobra"
)

func NewCmdCreateNodeGroup() *cobra.Command {
	opts := options.NewNodeGroupCreateConfig()
	cmd := &cobra.Command{
		Use: api.ResourceNameNodeGroup,
		Aliases: []string{
			api.ResourceTypeNodeGroup,
			api.ResourceCodeNodeGroup,
			api.ResourceKindNodeGroup,
		},
		Short:             "Create a Kubernetes cluster NodeGroup for a given cloud provider",
		Example:           "pharmer create nodegroup -k <cluster_name>",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			err := opts.ValidateFlags(cmd, args)
			if err != nil {
				term.Fatalln(err)
			}

			store.SetProvider(cmd, opts.Owner)

			cluster, err := store.StoreProvider.Clusters().Get(opts.ClusterName)
			if err != nil {
				term.Fatalln(err)
			}

			cm, err := cloud.GetCloudManager(cluster)
			if err != nil {
				term.Fatalln(err)
			}

			err = cloud.CreateMachineSets(cm, opts)
			if err != nil {
				term.Fatalln(err)
			}
		},
	}
	opts.AddFlags(cmd.Flags())

	return cmd
}
