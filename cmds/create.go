package cmds

import (
	cpCmd "github.com/pharmer/pharmer/cloud/cmds"
	credCmd "github.com/pharmer/pharmer/cmds/credential/cmds"
	"github.com/spf13/cobra"
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
	cmd.AddCommand(credCmd.NewCmdCreateCredential())

	return cmd
}
