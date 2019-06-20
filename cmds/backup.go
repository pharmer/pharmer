package cmds

import (
	cpCmd "github.com/pharmer/pharmer/cmds/cloud"
	"github.com/spf13/cobra"
)

func newCmdBackup() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "backup",
		DisableAutoGenTag: true,
	}
	cmd.AddCommand(cpCmd.NewCmdBackup())

	return cmd
}
