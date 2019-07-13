package credential

import (
	"io"

	"github.com/appscode/go/term"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cloudapi "pharmer.dev/cloud/pkg/apis/cloud/v1"
	"pharmer.dev/pharmer/cmds/credential/options"
	"pharmer.dev/pharmer/store"
	"pharmer.dev/pharmer/utils/printer"
)

func NewCmdGetCredential(out io.Writer) *cobra.Command {
	opts := options.NewCredentialGetConfig()
	cmd := &cobra.Command{
		Use: cloudapi.ResourceNameCredential,
		Aliases: []string{
			cloudapi.ResourceTypeCredential,
			cloudapi.ResourceCodeCredential,
			cloudapi.ResourceKindCredential,
		},
		Short:             "List cloud Credentials",
		Example:           `pharmer get credential`,
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.ValidateFlags(cmd, args); err != nil {
				term.Fatalln(err)
			}

			storeProvider, err := store.GetStoreProvider(cmd)
			if err != nil {
				term.Fatalln(err)
			}

			err = RunGetCredential(storeProvider.Credentials(), opts, out)
			term.ExitOnError(err)
		},
	}
	opts.AddFlags(cmd.Flags())

	return cmd
}

func RunGetCredential(credStore store.CredentialStore, opts *options.CredentialGetConfig, out io.Writer) error {
	rPrinter, err := printer.NewPrinter(opts.Output)
	if err != nil {
		return err
	}

	w := printer.GetNewTabWriter(out)

	credentials, err := getCredentialList(credStore, opts.Credentials)
	if err != nil {
		return err
	}
	for _, credential := range credentials {
		if err := rPrinter.PrintObj(credential, w); err != nil {
			return err
		}
		err = printer.PrintNewline(w)
		if err != nil {
			return err
		}
	}

	return w.Flush()
}

func getCredentialList(credStore store.CredentialStore, args []string) (credentialList []*cloudapi.Credential, err error) {
	if len(args) != 0 {
		for _, arg := range args {
			credential, er2 := credStore.Get(arg)
			if er2 != nil {
				return nil, er2
			}
			credentialList = append(credentialList, credential)
		}

	} else {
		credentialList, err = credStore.List(metav1.ListOptions{})
		if err != nil {
			return
		}
	}
	return
}
