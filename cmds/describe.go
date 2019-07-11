package cmds

import (
	"io"

	cpCmd "github.com/pharmer/pharmer/cmds/cloud"
	"github.com/spf13/cobra"
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
