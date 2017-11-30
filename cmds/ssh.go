package cmds

import (
	cpCmd "github.com/pharmer/pharmer/cloud/cmds"
	"github.com/spf13/cobra"
)

func newCmdSSH() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "ssh",
		DisableAutoGenTag: true,
	}
	cmd.AddCommand(cpCmd.NewCmdSSH())

	return cmd
}
