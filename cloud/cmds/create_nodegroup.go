package cmds

import (
	"github.com/appscode/go/term"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/cloud/cmds/options"
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

			CreateMachineSets(opts)
		},
	}
	opts.AddFlags(cmd.Flags())

	return cmd
}

func CreateMachineSets(opts *options.NodeGroupCreateConfig) {
	cluster, err := cloud.Get(opts.ClusterName, opts.Owner)
	term.ExitOnError(err)
	for sku, count := range opts.Nodes {
		err := cloud.CreateMachineSet(cluster, opts.Owner, api.RoleNode, sku, api.NodeType(opts.NodeType), int32(count), opts.SpotPriceMax)
		term.ExitOnError(err)
	}
}
