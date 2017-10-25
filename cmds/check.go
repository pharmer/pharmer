package cmds

import (
	cpCmd "github.com/appscode/pharmer/cloud/cmds"
	"github.com/spf13/cobra"
)

func newCmdCheck() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "check",
		DisableAutoGenTag: true,
	}

	// Cloud
	cmd.AddCommand(cpCmd.NewCmdCheckCluster())
	return cmd
}
