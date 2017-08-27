package cmds

import (
	"github.com/spf13/cobra"
)

func NewCmdCredential() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "credential",
		Short:             "Manage cloud provider credentials",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
	cmd.AddCommand(NewCmdImport())
	cmd.AddCommand(NewCmdIssue())
	cmd.AddCommand(NewCmdDelete())
	cmd.AddCommand(NewCmdGet())
	return cmd
}
