package cmds

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/tabwriter"
	"github.com/pharmer/pharmer/config"
	"github.com/spf13/cobra"
)

func newCmdGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "get-contexts",
		Short:             "List available contexts",
		Example:           "Pharmer config get-contexts",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				cmd.Help()
				os.Exit(1)
			}

			w := new(tabwriter.Writer)
			w.Init(os.Stdout, 0, 8, 0, '\t', 0)
			fmt.Fprintln(w, "NAME\tStore")
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
