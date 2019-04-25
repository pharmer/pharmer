package cmds

import (
	"github.com/appscode/go/term"
	cloudapi "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/config"
	"github.com/pharmer/pharmer/credential/cmds/options"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

func NewCmdDeleteCredential() *cobra.Command {
	opts := options.NewCredentialDeleteConfig()
	cmd := &cobra.Command{
		Use: cloudapi.ResourceNameCredential,
		Aliases: []string{
			cloudapi.ResourceTypeCredential,
			cloudapi.ResourceCodeCredential,
			cloudapi.ResourceKindCredential,
		},
		Short:             "Delete  credential object",
		Example:           `pharmer delete credential`,
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.ValidateFlags(cmd, args); err != nil {
				term.Fatalln(err)
			}

			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			term.ExitOnError(err)

			ctx := cloud.NewContext(context.Background(), cfg, config.GetEnv(cmd.Flags()))

			for _, cred := range opts.Credentials {
				err := cloud.Store(ctx).Owner(opts.Owner).Credentials().Delete(cred)
				term.ExitOnError(err)
			}
		},
	}
	opts.AddFlags(cmd.Flags())

	return cmd
}
