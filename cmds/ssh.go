package cmds

import (
	cpCmd "github.com/appscode/pharmer/cloud/cmds"
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
