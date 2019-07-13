package cmds

import (
	"io"

	"github.com/spf13/cobra"
	cpCmd "pharmer.dev/pharmer/cmds/cloud"
	"pharmer.dev/pharmer/cmds/credential"
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
	cmd.AddCommand(credential.NewCmdEditCredential(out, outErr))

	return cmd
}
