package cmds

import (
	"flag"

	"github.com/appscode/go/term"
	"github.com/pharmer/pharmer/hack/pharmer-tools/cmds/options"
	"github.com/pharmer/pharmer/hack/pharmer-tools/providers"
	"github.com/spf13/cobra"
)

func NewCmdKubeSupport() *cobra.Command {
	opts := options.NewKubernetesData()
	cmd := &cobra.Command{
		Use:               "kubesupport",
		Short:             "Add kubernetes version support for a given or all cloud provider",
		Example:           "",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.ValidateFlags(cmd, args); err != nil {
				term.Fatalln(err)
			}
			err := providers.AddKubernetesSupport(opts)
			if err != nil {
				term.Fatalln(err)
			} else {
				term.Successln("Kubernetes support successfully added for ", opts.Provider)
			}
		},
	}
	cmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	opts.AddFlags(cmd.Flags())
	return cmd
}
