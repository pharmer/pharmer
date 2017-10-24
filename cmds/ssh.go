package cmds

import (
	cpCmd "github.com/appscode/pharmer/cloud/cmds"
	"github.com/spf13/cobra"
)

func newCmdSSH() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "ssh",
		DisableAutoGenTag: true,
		Run:               func(cmd *cobra.Command, args []string) {},
	}
	cmd.AddCommand(cpCmd.NewCmdSSH())

	return cmd
}
