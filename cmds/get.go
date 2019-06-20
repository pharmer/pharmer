package cmds

import (
	"io"

	cpCmd "github.com/pharmer/pharmer/cmds/cloud"
	"github.com/pharmer/pharmer/cmds/credential"
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
	cmd.AddCommand(credential.NewCmdGetCredential(out))

	return cmd
}
