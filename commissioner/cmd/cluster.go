package cmd

import (
	"github.com/spf13/cobra"
)

func NewCmdCluster() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Various cluster comissioning commands.",
	}
	cmd.AddCommand(NewCmdClusterCreate())
	cmd.AddCommand(NewCmdClusterNative())
	cmd.AddCommand(NewCmdClusterScale())
	cmd.AddCommand(NewCmdClusterConfig())
	cmd.AddCommand(NewCmdClusterDelete())
	cmd.AddCommand(NewCmdClusterNetworks())
	cmd.AddCommand(NewCmdClusterAddon())
	return cmd
}
