package cmds

import (
	"io"

	"github.com/spf13/cobra"
)

func NewCmdDescribe(out, errOut io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "describe",
		DisableAutoGenTag: true,
		Run:               func(cmd *cobra.Command, args []string) {},
	}

	cmd.AddCommand(NewCmdDescribeCluster(out, errOut))
	return cmd
}
