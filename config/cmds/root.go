package cmds

import "github.com/spf13/cobra"

func NewCmdConfig() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:               "config",
		Short:             "Pharmer configuration",
		Example:           "pharmer config view",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	rootCmd.AddCommand(newCmdView())
	rootCmd.AddCommand(newCmdGet())
	return rootCmd
}
