package cmds

import (
	credCmd "github.com/pharmer/pharmer/credential/cmds"
	"github.com/spf13/cobra"
)

func newCmdIssue() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "issue",
		DisableAutoGenTag: true,
		Run:               func(cmd *cobra.Command, args []string) {},
	}

	// Credential
	cmd.AddCommand(credCmd.NewCmdIssue())

	return cmd
}
