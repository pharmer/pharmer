package cmds

import (
	cpCmd "github.com/pharmer/pharmer/cmds/cloud"
	"github.com/pharmer/pharmer/cmds/credential"
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
	cmd.AddCommand(credential.NewCmdCreateCredential())

	return cmd
}
