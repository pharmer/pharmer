package cmd

import (
	"github.com/spf13/cobra"
)

func NewCmdCommisioner() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "commissioner",
		Short: "Various commissioning commands.",
		DisableAutoGenTag: true,
	}

	rootCmd.AddCommand(NewCmdClusterCreate())
	rootCmd.AddCommand(NewCmdClusterNative())
	rootCmd.AddCommand(NewCmdClusterScale())
	rootCmd.AddCommand(NewCmdClusterConfig())
	rootCmd.AddCommand(NewCmdClusterDelete())
	rootCmd.AddCommand(NewCmdClusterNetworks())
	rootCmd.AddCommand(NewCmdClusterAddon())

	return rootCmd
}
