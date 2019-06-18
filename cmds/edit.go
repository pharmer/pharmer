package cmds

import (
	"io"

	cpCmd "github.com/pharmer/pharmer/cloud/cmds"
	"github.com/pharmer/pharmer/cmds/credential"
	"github.com/spf13/cobra"
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
