package cmds

import (
	"github.com/spf13/cobra"
	inCmd "pharmer.dev/pharmer/inspector/cmds"
)

func NewCmdInspector() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "inspect",
		Short:             `Inspect cluster for conformance`,
		DisableAutoGenTag: true,
	}
	cmd.AddCommand(inCmd.NewCmdInspectCluster())
	return cmd
}
