package credential

import (
	"github.com/appscode/go/term"
	cloudapi "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	"github.com/pharmer/pharmer/cmds/credential/options"
	"github.com/pharmer/pharmer/store"
	"github.com/spf13/cobra"
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

			storeProvider, err := store.GetStoreProvider(cmd, opts.Owner)
			term.ExitOnError(err)

			err = runDeleteCredentialCmd(storeProvider.Credentials(), opts)

		},
	}
	opts.AddFlags(cmd.Flags())

	return cmd
}
func runDeleteCredentialCmd(credStore store.CredentialStore, opts *options.CredentialDeleteConfig) error {
	for _, cred := range opts.Credentials {
		err := credStore.Delete(cred)
		if err != nil {
			return err
		}
	}
	return nil
}
