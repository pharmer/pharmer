package cmds

import (
	"io"

	"github.com/spf13/cobra"
	cpCmd "pharmer.dev/pharmer/cmds/cloud"
)

func newCmdDescribe(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "describe",
		DisableAutoGenTag: true,
		Run:               func(cmd *cobra.Command, args []string) {},
	}

	cmd.AddCommand(cpCmd.NewCmdDescribeCluster(out))
	return cmd
}
