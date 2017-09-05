package cmds

import (
	gtx "context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/appscode/go/log"
	"github.com/appscode/pharmer/config"
	"github.com/appscode/pharmer/credential"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewCmdGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "get",
		Short:             "List cloud credentials",
		Example:           `pharmer credential list`,
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				fmt.Fprintf(os.Stderr, "No argument is supported, found %d", len(args))
				cmd.Help()
				os.Exit(1)
			}

			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				log.Fatalln(err)
			}
			store := config.NewStoreProvider(gtx.TODO(), cfg)
			creds, err := store.Credentials().List(metav1.ListOptions{})
			if err != nil {
				log.Fatalln(err)
			}

			w := new(tabwriter.Writer)
			w.Init(os.Stdout, 0, 8, 0, '\t', 0)
			fmt.Fprintln(w, "NAME\tProvider\tData")
			for _, c := range creds {
				spec := credential.CommonSpec(c.Spec)
				fmt.Fprintf(w, "%s\t%s\t%s\n", c.Name, spec.Provider, spec.String())
			}
			w.Flush()
		},
	}
	return cmd
}
