package cmds

import (
	"github.com/spf13/cobra"
)

func NewCmdDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "delete",
		Short:             "Delete a cloud credential",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {

		},
	}
	return cmd
}
