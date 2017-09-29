package cmds

import (
	"context"
	"strings"

	"github.com/appscode/go-term"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/config"
	cc "github.com/appscode/pharmer/credential/cloud"
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
		Short:             "Issue credential for cc providers Azure and Google Cloud",
		Example:           `pharmer issue credential mycred`,
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				term.Fatalln("Missing credential name.")
			}
			if len(args) > 1 {
				term.Fatalln("Multiple credential name provided.")
			}

			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				term.Fatalln(err)
			}
			ctx := cloud.NewContext(context.Background(), cfg, config.GetEnv(cmd.Flags()))

			provider, _ := cmd.Flags().GetString("provider")
			if provider == "GoogleCloud" {
				cc.IssueGCECredential(args[0])
			} else if strings.ToLower(provider) == "azure" {
				cred, err := cc.IssueAzureCredential(args[0])
				if err != nil {
					term.Fatalln(err)
				}
				_, err = cloud.Store(ctx).Credentials().Create(cred)
				if err != nil {
					term.Fatalln(err)
				}
			}
			term.Successln("Credential issued successfully!")
		},
	}

	cmd.Flags().StringP("provider", "p", "", "Name of the Cloud provider")
	return cmd
}
