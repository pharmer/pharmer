package cmds

import (
	"github.com/spf13/cobra"
	cpCmd "pharmer.dev/pharmer/cmds/cloud"
)

func newCmdUse() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "use",
		DisableAutoGenTag: true,
	}
	cmd.AddCommand(cpCmd.NewCmdUse())

	return cmd
}
