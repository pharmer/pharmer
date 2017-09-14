package cmds

import (
	"io"

	"github.com/appscode/log"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/config"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
	"gopkg.in/AlecAivazis/survey.v1"
	"github.com/appscode/pharmer/data/files"
	"fmt"
)

func NewCmdCreateCredential(out, errOut io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "credential",
		Short:             "Create  credential object",
		Example:           `pharmer create credential`,
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				log.Fatalln(err)
			}
			ctx := cloud.NewContext(context.Background(), cfg)

			runCreateCredential(ctx, cmd, out, errOut, args)
		},
	}

	cmd.Flags().StringP("provider", "p", "", "Name of the Cloud provider")

	return cmd
}

func runCreateCredential(ctx context.Context, cmd *cobra.Command, out, errOut io.Writer, args []string) error {
	// Get Cloud provider
	provider, _ := cmd.Flags().GetString("provider")
	if provider == "" {

		prompt := &survey.Select{
			Message: "Choose a Cloud provider:",
			Options: files.CredentialProviders().List(),
		}
		survey.AskOne(prompt, &provider, nil)
	}

	fmt.Println("--- ", provider)
	return nil
}

