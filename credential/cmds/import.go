package cmds

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/appscode/go-term"
	"github.com/appscode/log"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/config"
	"github.com/appscode/pharmer/data/files"
	"github.com/appscode/pharmer/storage"
	"github.com/appscode/pharmer/storage/providers/vfs"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			cred := api.Credential{
				ObjectMeta: api.ObjectMeta{
					Name:              args[0],
					CreationTimestamp: metav1.Time{Time: time.Now()},
				},
				Spec: api.CredentialSpec{
					Provider: provider,
					Data:     map[string]string{},
				},
			}
			api.AssignTypeKind(&cred)

			cf, _ := files.GetCredentialFormat(provider)
			for _, f := range cf.Fields {
				if f.Input == "password" {
					cred.Spec.Data[f.JSON] = term.ReadMasked(f.Label)
				} else {
					cred.Spec.Data[f.JSON] = term.Read(f.Label)
				}
			}

			b, _ := json.MarshalIndent(&cred, "", "  ")
			fmt.Println(string(b))

			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				log.Errorln(err)
			}
			fmt.Println(cfg)

			store, err := storage.GetProvider(vfs.UID, context.TODO(), cfg)
			if err != nil {
				log.Errorln(err)
			}
			_, err = store.Credentials().Create(&cred)
			if err != nil {
				log.Errorln(err)
			}
		},
	}
	return cmd
}
