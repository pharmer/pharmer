package cmds

import (
	"github.com/appscode/go/log"
	"github.com/appscode/go/term"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/cloud/cmds/options"
	"github.com/pharmer/pharmer/store"
	"github.com/spf13/cobra"
)

func NewCmdApply() *cobra.Command {
	opts := options.NewApplyConfig()
	cmd := &cobra.Command{
		Use:               "apply",
		Short:             "Apply changes",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.ValidateFlags(cmd, args); err != nil {
				term.Fatalln(err)
			}

			err := store.SetProvider(cmd, opts.Owner)
			if err != nil {
				term.Fatalln(err)
			}

			acts, err := cloud.Apply(opts)
			if err != nil {
				log.Fatalln(err)
			}
			for _, a := range acts {
				log.Infoln(a.Action, a.Resource, a.Message)
			}
		},
	}
	opts.AddFlags(cmd.Flags())
	return cmd
}
