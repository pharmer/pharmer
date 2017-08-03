package cmd

import (
	"github.com/spf13/cobra"
)

func NewCmdCommisioner() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "comissioner",
		Short: "Various comissioning commands.",
	}
	rootCmd.AddCommand(NewCmdCluster())

	return rootCmd
}
