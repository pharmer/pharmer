package cmds

import (
	"github.com/appscode/go/flags"
	"github.com/spf13/cobra"
)

func NewCmdCredentialDelete() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:               "delete",
		Short:             "Delete a cloud credential",
		Example:           `appctl credential delete --name="xyz"`,
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			flags.EnsureRequiredFlags(cmd, "name")
			// deleteCredential(name)
		},
	}
	cmd.Flags().StringVarP(&name, "name", "n", "", "credential name")
	return cmd
}
