package cmds

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/appscode/go-term"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/data/files"
	"github.com/spf13/cobra"
)

func NewCmdImport() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "import",
		Short:             "Import cloud credentials into Pharmer",
		Example:           `pharmer credential import -p aws mycred`,
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				fmt.Fprintf(os.Stderr, "You can only specify one argument, found %d", len(args))
				cmd.Help()
				os.Exit(1)
			}

			_, provider := term.List(files.CredentialProviders().List())
			c := api.Credential{
				ObjectMeta: api.ObjectMeta{
					Name: args[0],
				},
				Spec: api.CredentialSpec{
					Provider: provider,
					Data:     map[string]string{},
				},
			}
			api.AssignTypeKind(&c)
			cf, _ := files.GetCredentialFormat(provider)
			for _, f := range cf.Fields {
				if f.Input == "password" {
					c.Spec.Data[f.JSON] = term.ReadMasked(f.Label)
				} else {
					c.Spec.Data[f.JSON] = term.Read(f.Label)
				}
			}

			b, _ := json.MarshalIndent(&c, "", "  ")
			fmt.Println(string(b))
		},
	}
	return cmd
}
