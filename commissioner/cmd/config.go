package cmd

import (
	term "github.com/appscode/go-term"
	"github.com/appscode/go/flags"
	"github.com/appscode/pharmer/commissioner"
	"github.com/spf13/cobra"
)

func NewCmdClusterConfig() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Cluster commissioning config",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			flags.SetLogLevel(4)
			if len(args) > 0 {
				name = args[0]
			} else {
				term.Fatalln("missing cluster name")
			}
			c, err := commissioner.NewComissionar("", name)
			term.ExitOnError(err)
			c.InstallKubeConfig()
		},
	}
	return cmd
}
