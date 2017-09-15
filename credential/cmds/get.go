package cmds

import (
	"io"

	"github.com/appscode/go-term"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/cloud/printer"
	"github.com/appscode/pharmer/config"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewCmdGetCredential(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use: api.ResourceNameCredential,
		Aliases: []string{
			api.ResourceTypeCredential,
			api.ResourceCodeCredential,
			api.ResourceKindCredential,
		},
		Short:             "List cloud Credentials",
		Example:           `pharmer get credential`,
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			term.ExitOnError(err)

			ctx := cloud.NewContext(context.Background(), cfg)
			RunGetCredential(ctx, cmd, out, args)
		},
	}
	cmd.Flags().StringP("output", "o", "", "Output format. One of: json|yaml|wide")
	return cmd
}

func RunGetCredential(ctx context.Context, cmd *cobra.Command, out io.Writer, args []string) error {

	rPrinter, err := printer.NewPrinter(cmd)
	if err != nil {
		return err
	}

	w := printer.GetNewTabWriter(out)

	credentials, err := getCredentialList(ctx, args)
	if err != nil {
		return err
	}
	for _, credential := range credentials {
		if err := rPrinter.PrintObj(credential, w); err != nil {
			return err
		}
		if rPrinter.IsGeneric() {
			printer.PrintNewline(w)
		}
	}

	w.Flush()
	return nil
}

func getCredentialList(ctx context.Context, args []string) (credentialList []*api.Credential, err error) {
	if len(args) != 0 {
		for _, arg := range args {
			credential, er2 := cloud.Store(ctx).Credentials().Get(arg)
			if er2 != nil {
				return nil, er2
			}
			credentialList = append(credentialList, credential)
		}

	} else {
		credentialList, err = cloud.Store(ctx).Credentials().List(metav1.ListOptions{})
		if err != nil {
			return
		}
	}
	return
}
