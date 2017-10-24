package cmds

import (
	cpCmd "github.com/appscode/pharmer/cloud/cmds"
	"github.com/spf13/cobra"
)

func newCmdUse() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "use",
		DisableAutoGenTag: true,
		Run:               func(cmd *cobra.Command, args []string) {},
	}
	cmd.AddCommand(cpCmd.NewCmdUse())

	return cmd
}
