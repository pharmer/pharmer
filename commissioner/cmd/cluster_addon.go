package cmd

import (
	"fmt"
	"os"

	term "github.com/appscode/go-term"
	"github.com/appscode/go/flags"
	"github.com/appscode/pharmer/commissioner"
	"github.com/spf13/cobra"
)

func NewCmdClusterAddon() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "addon",
		Short: "Cluster commisioning addon setup",
		Run: func(cmd *cobra.Command, args []string) {
			flags.SetLogLevel(4)
			if len(args) > 0 {
				name = args[0]
			} else {
				term.Fatalln("missing cluster name")
			}
			c, err := commissioner.NewComissionar("", name)
			term.ExitOnError(err)
			err = c.AddonSetup()
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
		},
	}
	return cmd
}
