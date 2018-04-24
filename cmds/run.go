package cmds

import (
	conCmd "github.com/pharmer/pharmer/controller/cmds"
	"github.com/spf13/cobra"
)

func newCmdRun() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "run",
		DisableAutoGenTag: true,
		Run:               func(cmd *cobra.Command, args []string) {},
	}

	// Cloud
	// Controller
	cmd.AddCommand(conCmd.NewCmdRunController())

	return cmd
}
