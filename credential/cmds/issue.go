package cmds

import (
	"fmt"

	"github.com/appscode/go-term"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/credential/cloud"
	"github.com/spf13/cobra"
)

func NewCmdIssue() *cobra.Command {
	cmd := &cobra.Command{
		Use: api.ResourceNameCredential,
		Aliases: []string{
			api.ResourceTypeCredential,
			api.ResourceCodeCredential,
			api.ResourceKindCredential,
		},
		Short:             "Issue credential for cloud providers Azure and Google Cloud",
		Example:           `pharmer issue credential mycred`,
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				term.Fatalln(fmt.Sprintf("You can only specify one argument, found %d", len(args)))
			}

			provider, _ := cmd.Flags().GetString("provider")
			if provider == "GoogleCloud" {
				cloud.IssueGCECredential(args[0])
			} else if provider == "Azure" {
				cloud.IssueAzureCredential(args[0])
			}
			term.Successln("Credential issued successfully!")
		},
	}

	cmd.Flags().StringP("provider", "p", "", "Name of the Cloud provider")
	return cmd
}
