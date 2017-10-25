package cmds

import (
	cpCmd "github.com/appscode/pharmer/cloud/cmds"
	"github.com/spf13/cobra"
)

func newCmdUse() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "use",
		DisableAutoGenTag: true,
	}
	cmd.AddCommand(cpCmd.NewCmdUse())

	return cmd
}
