package cmds

import (
	"github.com/spf13/cobra"
)

func NewCmdCredential() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:               "credential",
		Short:             "Manage cloud provider credentials",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
	rootCmd.AddCommand(NewCmdCredentialCreate())
	rootCmd.AddCommand(NewCmdCredentialUpdate())
	rootCmd.AddCommand(NewCmdCredentialDelete())
	rootCmd.AddCommand(NewCmdCredentialList())
	return rootCmd
}
