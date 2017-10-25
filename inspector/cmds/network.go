package cmds

import (
	"fmt"
	"os"

	"github.com/appscode/go-term"
	"github.com/appscode/go/flags"
	"github.com/appscode/pharmer/inspector"
	"github.com/spf13/cobra"
)

func NewCmdNetworks() *cobra.Command {
	var name, kubeconfig, rsafile string
	cmd := &cobra.Command{
		Use:               "network",
		Short:             "Cluster commissioning network check",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			flags.EnsureRequiredFlags(cmd, "rsa-private-key")
			flags.SetLogLevel(4)
			if len(args) > 0 {
				name = args[0]
			} else {
				term.Fatalln("missing cluster name")
			}
			c, err := inspector.New(kubeconfig, name)
			term.ExitOnError(err)
			err = c.LoadSSHKey(rsafile)
			term.Fatalln(err)

			err = c.NetworkCheck()
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
		},
	}
	cmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Cluster configuration file")
	cmd.Flags().StringVar(&rsafile, "rsa-private-key", "", "ras private key file")
	return cmd
}
