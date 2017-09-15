package cmds

import (
	cpCmd "github.com/appscode/pharmer/cloud/cmds"
	credCmd "github.com/appscode/pharmer/credential/cmds"
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

	// Credential
	cmd.AddCommand(credCmd.NewCmdCreateCredential())

	return cmd
}
