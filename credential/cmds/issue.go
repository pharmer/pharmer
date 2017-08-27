package cmds

import (
	"fmt"
	"os"

	"github.com/appscode/go-term"
	"github.com/appscode/pharmer/credential/cloud"
	"github.com/spf13/cobra"
)

func NewCmdIssue() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "issue",
		Short:             "Issue credential for cloud providers Azure and Google Cloud",
		Example:           `pharmer credential issue mycred`,
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				fmt.Fprintf(os.Stderr, "You can only specify one argument, found %d", len(args))
				cmd.Help()
				os.Exit(1)
			}

			_, provider := term.List([]string{"Azure", "GoogleCloud"})
			if provider == "gce" {
				cloud.IssueGCECredential(args[0])
			} else if provider == "azure" {
				cloud.IssueAzureCredential(args[0])
			}
			term.Successln("Credential issued successfully!")
		},
	}
	return cmd
}
