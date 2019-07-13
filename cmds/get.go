package cmds

import (
	"io"

	"github.com/spf13/cobra"
	cpCmd "pharmer.dev/pharmer/cmds/cloud"
	"pharmer.dev/pharmer/cmds/credential"
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
	cmd.AddCommand(credential.NewCmdGetCredential(out))

	return cmd
}
