package cmds

import (
	"fmt"
	"os"

	"github.com/appscode/go-term"
	"github.com/appscode/go/flags"
	"github.com/appscode/pharmer/inspector"
	"github.com/spf13/cobra"
)

func NewCmdAddon() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:               "addon",
		Short:             "Cluster commissioning addon setup",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			flags.SetLogLevel(4)
			if len(args) > 0 {
				name = args[0]
			} else {
				term.Fatalln("missing cluster name")
			}
			c, err := inspector.New("", name)
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
