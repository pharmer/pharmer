package cmds

import (
	cpCmd "github.com/appscode/pharmer/cloud/cmds"
	"github.com/spf13/cobra"
)

func newCmdBackup() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "backup",
		DisableAutoGenTag: true,
		Run:               func(cmd *cobra.Command, args []string) {},
	}
	cmd.AddCommand(cpCmd.NewCmdBackup())

	return cmd
}
