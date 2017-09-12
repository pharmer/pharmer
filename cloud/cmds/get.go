package cmds

import (
	"io"

	"github.com/spf13/cobra"
)

func NewCmdGet(out, errOut io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "get",
		DisableAutoGenTag: true,
		Run:               func(cmd *cobra.Command, args []string) {},
	}

	cmd.AddCommand(NewCmdGetCluster(out, errOut))
	cmd.AddCommand(NewCmdGetNodeGroup(out, errOut))
	return cmd
}
