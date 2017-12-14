package cmds

import (
	"github.com/appscode/go/term"
	api "github.com/pharmer/pharmer/apis/v1alpha1"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/config"
	"github.com/pharmer/pharmer/credential/cmds/options"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

func NewCmdDeleteCredential() *cobra.Command {
	credConfig := options.NewCredentialDeleteConfig()
	cmd := &cobra.Command{
		Use: api.ResourceNameCredential,
		Aliases: []string{
			api.ResourceTypeCredential,
			api.ResourceCodeCredential,
			api.ResourceKindCredential,
		},
		Short:             "Delete  credential object",
		Example:           `pharmer delete credential`,
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if err := credConfig.ValidateCredentialDeleteFlags(cmd, args); err != nil {
				term.Fatalln(err)
			}

			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			term.ExitOnError(err)

			ctx := cloud.NewContext(context.Background(), cfg, config.GetEnv(cmd.Flags()))

			for _, cred := range credConfig.Credentials {
				err := cloud.Store(ctx).Credentials().Delete(cred)
				term.ExitOnError(err)
			}
		},
	}
	credConfig.AddCredentialDeleteFlags(cmd.Flags())

	return cmd
}
