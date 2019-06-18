package credential

import (
	"github.com/appscode/go/term"
	cloudapi "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	options2 "github.com/pharmer/pharmer/cmds/credential/options"
	"github.com/pharmer/pharmer/store"
	"github.com/spf13/cobra"
)

func NewCmdDeleteCredential() *cobra.Command {
	opts := options2.NewCredentialDeleteConfig()
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

			storeProvider, err := store.GetStoreProvider(cmd, opts.Owner)
			term.ExitOnError(err)

			for _, cred := range opts.Credentials {
				err := storeProvider.Credentials().Delete(cred)
				term.ExitOnError(err)
			}
		},
	}
	opts.AddFlags(cmd.Flags())

	return cmd
}
