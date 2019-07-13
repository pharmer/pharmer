package cmds

import (
	"github.com/spf13/cobra"
	cpCmd "pharmer.dev/pharmer/cmds/cloud"
	"pharmer.dev/pharmer/cmds/credential"
)

func newCmdCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "create",
		DisableAutoGenTag: true,
		Run:               func(cmd *cobra.Command, args []string) {},
	}

	// Cloud
	cmd.AddCommand(cpCmd.NewCmdCreateCluster())
	cmd.AddCommand(cpCmd.NewCmdCreateNodeGroup())

	// Credential
	cmd.AddCommand(credential.NewCmdCreateCredential())

	return cmd
}
