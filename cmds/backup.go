package cmds

import (
	"github.com/spf13/cobra"
	cpCmd "pharmer.dev/pharmer/cmds/cloud"
)

func newCmdBackup() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "backup",
		DisableAutoGenTag: true,
	}
	cmd.AddCommand(cpCmd.NewCmdBackup())

	return cmd
}
