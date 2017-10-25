package cmds

import (
	"fmt"
	"os"

	"github.com/appscode/go-term"
	"github.com/appscode/go/flags"
	"github.com/appscode/pharmer/inspector"
	"github.com/spf13/cobra"
)

func NewCmdNative() *cobra.Command {
	var name, kubeconfig string
	cmd := &cobra.Command{
		Use:               "native",
		Short:             "Cluster commissioning native check",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			//flags.EnsureRequiredFlags(cmd, "ku")
			flags.SetLogLevel(4)
			if len(args) > 0 {
				name = args[0]
			} else {
				term.Fatalln("missing cluster name")
			}
			c, err := inspector.New(kubeconfig, name)
			term.ExitOnError(err)
			err = c.NativeCheck()
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
		},
	}

	cmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Cluster configuration file")
	return cmd
}
