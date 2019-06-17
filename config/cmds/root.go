package cmds

import (
	"github.com/appscode/go/term"
	"github.com/spf13/cobra"
)

func NewCmdConfig() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:               "config",
		Short:             "Pharmer configuration",
		Example:           "pharmer config view",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			term.ExitOnError(err)
		},
	}

	rootCmd.AddCommand(newCmdView())
	rootCmd.AddCommand(newCmdGet())
	rootCmd.AddCommand(newCmdCreate())
	return rootCmd
}
