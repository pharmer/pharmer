package cmds

import (
	"github.com/spf13/cobra"
)

func NewCmdCredential() *cobra.Command {
	credentialRoot := &cobra.Command{
		Use:               "credential",
		Short:             "Manage cloud provider credentials",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
	credentialRoot.AddCommand(NewCmdCredentialCreate())
	credentialRoot.AddCommand(NewCmdCredentialUpdate())
	credentialRoot.AddCommand(NewCmdCredentialDelete())
	credentialRoot.AddCommand(NewCmdCredentialList())
	return credentialRoot
}
