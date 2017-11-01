package cmds

import (
	cpCmd "github.com/appscode/pharmer/cloud/cmds"
	credCmd "github.com/appscode/pharmer/credential/cmds"
	"github.com/spf13/cobra"
	"io"
)

func newCmdEdit(out, outErr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "edit",
		DisableAutoGenTag: true,
		Run:               func(cmd *cobra.Command, args []string) {},
	}

	// Cloud
	cmd.AddCommand(cpCmd.NewCmdEditCluster(out, outErr))
	cmd.AddCommand(cpCmd.NewCmdEditNodeGroup(out, outErr))

	// Credential
	cmd.AddCommand(credCmd.NewCmdEditCredential(out, outErr))

	return cmd
}
