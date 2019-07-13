package cmds

import (
	"github.com/spf13/cobra"
	cpCmd "pharmer.dev/pharmer/cmds/cloud"
)

func newCmdSSH() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "ssh",
		DisableAutoGenTag: true,
	}
	cmd.AddCommand(cpCmd.NewCmdSSH())

	return cmd
}
