package cmds

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/appscode/go/term"
	"github.com/spf13/cobra"
	"pharmer.dev/pharmer/config"
)

func newCmdGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "get-contexts",
		Short:             "List available contexts",
		Example:           "Pharmer config get-contexts",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				err := cmd.Help()
				if err != nil {
					term.Fatalln(err)
				}
				os.Exit(1)
			}

			w := new(tabwriter.Writer)
			w.Init(os.Stdout, 0, 8, 0, '\t', 0)
			_, err := fmt.Fprintln(w, "NAME\tStore")
			if err != nil {
				term.Fatalln(err)
			}
			files, err := ioutil.ReadDir(config.ConfigDir(cmd.Flags()))
			if err != nil {
				return err
			}
			for _, f := range files {
				cfg, err := config.LoadConfig(filepath.Join(config.ConfigDir(cmd.Flags()), f.Name()))
				if err != nil {
					return err
				}
				fmt.Fprintf(w, "%s\t%s\n", cfg.Context, cfg.GetStoreType())
			}
			return w.Flush()
		},
	}
	return cmd
}
