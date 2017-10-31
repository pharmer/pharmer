package cmds

import (
	cpCmd "github.com/appscode/pharmer/cloud/cmds"
	"github.com/spf13/cobra"
	"io"
)

func newCmdEdit(out, outErr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "edit",
		DisableAutoGenTag: true,
		Run:               func(cmd *cobra.Command, args []string) {},
	}

	// Cloud
	cmd.AddCommand(cpCmd.NewCmdEditCluster(out, outErr))
	cmd.AddCommand(cpCmd.NewCmdEditNodeGroup(out, outErr))
	return cmd
}
