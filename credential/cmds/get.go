package cmds

import (
	"io"

	"github.com/pharmer/pharmer/store"

	"github.com/appscode/go/term"
	cloudapi "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	"github.com/pharmer/pharmer/credential/cmds/options"
	"github.com/pharmer/pharmer/utils/printer"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

			store.SetProvider(cmd, opts.Owner)

			err := RunGetCredential(opts, out)
			term.ExitOnError(err)
		},
	}
	opts.AddFlags(cmd.Flags())

	return cmd
}

func RunGetCredential(opts *options.CredentialGetConfig, out io.Writer) error {
	rPrinter, err := printer.NewPrinter(opts.Output)
	if err != nil {
		return err
	}

	w := printer.GetNewTabWriter(out)

	credentials, err := getCredentialList(opts.Credentials, opts.Owner)
	if err != nil {
		return err
	}
	for _, credential := range credentials {
		if err := rPrinter.PrintObj(credential, w); err != nil {
			return err
		}
		printer.PrintNewline(w)
	}

	w.Flush()
	return nil
}

func getCredentialList(args []string, owner string) (credentialList []*cloudapi.Credential, err error) {
	if len(args) != 0 {
		for _, arg := range args {
			credential, er2 := store.StoreProvider.Credentials().Get(arg)
			if er2 != nil {
				return nil, er2
			}
			credentialList = append(credentialList, credential)
		}

	} else {
		credentialList, err = store.StoreProvider.Credentials().List(metav1.ListOptions{})
		if err != nil {
			return
		}
	}
	return
}
