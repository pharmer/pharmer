package cmds

import (
	"github.com/spf13/cobra"
)

func NewCmdDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "delete",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {},
	}

	cmd.AddCommand(NewCmdDeleteCluster())
	cmd.AddCommand(NewCmdDeleteNodeGroup())
	return cmd
}
