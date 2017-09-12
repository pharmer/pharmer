package cmds

import (
	"io"

	"github.com/spf13/cobra"
)

func NewCmdGet(out, err io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "get",
		DisableAutoGenTag: true,
		Run:               func(cmd *cobra.Command, args []string) {},
	}

	cmd.AddCommand(NewCmdGetCluster(out, err))
	return cmd
}
