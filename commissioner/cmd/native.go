package cmd

import (
	"fmt"
	"os"

	term "github.com/appscode/go-term"
	"github.com/appscode/go/flags"
	"github.com/appscode/pharmer/commissioner"
	"github.com/spf13/cobra"
)

func NewCmdClusterNative() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:               "native",
		Short:             "Cluster commissioning native check",
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
			err = c.NativeCheck()
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
			/*_, err := installer.Create(&req)
			if err != nil {
				errors.Exit(err)
			}*/
		},
	}
	return cmd
}
