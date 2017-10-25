package cmds

import (
	v "github.com/appscode/go/version"
	inCmd "github.com/appscode/pharmer/inspector/cmds"
	"github.com/spf13/cobra"
)

func NewCmdInspector() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "inspect",
		Short:             `Inspect cluster for conformance`,
		DisableAutoGenTag: true,
	}
	cmd.AddCommand(inCmd.NewCmdNative())
	cmd.AddCommand(inCmd.NewCmdNetworks())
	cmd.AddCommand(inCmd.NewCmdAddon())
	cmd.AddCommand(v.NewCmdVersion())
	return cmd
}
