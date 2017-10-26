package cmds

import (
	inCmd "github.com/appscode/pharmer/inspector/cmds"
	"github.com/spf13/cobra"
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
