package cmd

import (
	"fmt"
	"os"

	term "github.com/appscode/go-term"
	"github.com/appscode/go/flags"
	"github.com/appscode/pharmer/commissioner"
	"github.com/spf13/cobra"
)

func NewCmdClusterScale() *cobra.Command {
	var provider string
	var name string
	cmd := &cobra.Command{
		Use:               "scale",
		Short:             "Cluster create commissioning",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			flags.SetLogLevel(4)
			flags.EnsureRequiredFlags(cmd, "provider")
			c, err := commissioner.NewComissionar(provider, name)
			term.ExitOnError(err)
			err = c.LoadKubeClient()
			term.ExitOnError(err)
			err = c.ClusterScale()
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}

		},
	}
	cmd.Flags().StringVarP(&provider, "provider", "p", "", "Give the provider Name")
	cmd.Flags().StringVarP(&name, "name", "n", "", "Give the cluster Name")
	return cmd
}
