package cmds

import (
	"github.com/appscode/go/term"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/cloud/cmds/options"
	"github.com/spf13/cobra"
)

func NewCmdDeleteNodeGroup() *cobra.Command {
	opts := options.NewNodeGroupDeleteConfig()
	cmd := &cobra.Command{
		Use: api.ResourceNameNodeGroup,
		Aliases: []string{
			api.ResourceTypeNodeGroup,
			api.ResourceCodeNodeGroup,
			api.ResourceKindNodeGroup,
		},
		Short:             "Delete a Kubernetes cluster NodeGroup",
		Example:           "pharmer delete nodegroup -k <cluster_name>",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.ValidateFlags(cmd, args); err != nil {
				term.Fatalln(err)
			}

			nodeGroups, err := GetMachineSetList(opts.ClusterName, args...)
			term.ExitOnError(err)

			for _, ng := range nodeGroups {
				err := cloud.DeleteMachineSet(opts.ClusterName, ng.Name)
				term.ExitOnError(err)
			}
		},
	}
	opts.AddFlags(cmd.Flags())

	return cmd
}
