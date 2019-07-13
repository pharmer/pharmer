package cmds

import (
	"github.com/spf13/cobra"
	cpCmd "pharmer.dev/pharmer/cmds/cloud"
	"pharmer.dev/pharmer/cmds/credential"
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
	cmd.AddCommand(credential.NewCmdDeleteCredential())

	return cmd
}
