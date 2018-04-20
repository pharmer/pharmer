package cmds

import (
	"io"
	cpCmd "github.com/pharmer/pharmer/cloud/cmds"
	credCmd "github.com/pharmer/pharmer/credential/cmds"
	"github.com/spf13/cobra"
)

func newCmdGet(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "get",
		DisableAutoGenTag: true,
		Run:               func(cmd *cobra.Command, args []string) {},
	}

	// Cloud
	cmd.AddCommand(cpCmd.NewCmdGetCluster(out))
	cmd.AddCommand(cpCmd.NewCmdGetNodeGroup(out))

	// Credential
	cmd.AddCommand(credCmd.NewCmdGetCredential(out))

	return cmd
}
