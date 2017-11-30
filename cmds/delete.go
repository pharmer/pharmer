package cmds

import (
	cpCmd "github.com/pharmer/pharmer/cloud/cmds"
	credCmd "github.com/pharmer/pharmer/credential/cmds"
	"github.com/spf13/cobra"
)

func newCmdDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "delete",
		DisableAutoGenTag: true,
		Run:               func(cmd *cobra.Command, args []string) {},
	}

	// Cloud
	cmd.AddCommand(cpCmd.NewCmdDeleteCluster())
	cmd.AddCommand(cpCmd.NewCmdDeleteNodeGroup())

	// Credential
	cmd.AddCommand(credCmd.NewCmdDeleteCredential())

	return cmd
}
