package cmd

import (
	term "github.com/appscode/go-term"
	"github.com/appscode/go/flags"
	"github.com/appscode/pharmer/commissioner"
	"github.com/spf13/cobra"
)

func NewCmdClusterDelete() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Cluster commissioning delete",
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
			c.ClusterDelete()
		},
	}
	return cmd
}
