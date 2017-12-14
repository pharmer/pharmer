package cmds

import (
	"context"

	"github.com/appscode/go/log"
	"github.com/appscode/go/term"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/cloud/cmds/options"
	"github.com/pharmer/pharmer/config"
	"github.com/spf13/cobra"
)

func NewCmdApply() *cobra.Command {
	applyConfig := options.NewApplyConfig()
	cmd := &cobra.Command{
		Use:               "apply",
		Short:             "Apply changes",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if err := applyConfig.ValidateApplyFlags(cmd, args); err != nil {
				term.Fatalln(err)
			}

			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				log.Fatalln(err)
			}
			ctx := cloud.NewContext(context.Background(), cfg, config.GetEnv(cmd.Flags()))

			acts, err := cloud.Apply(ctx, applyConfig.ClusterName, applyConfig.DryRun)
			if err != nil {
				log.Fatalln(err)
			}
			for _, a := range acts {
				log.Infoln(a.Action, a.Resource, a.Message)
			}
		},
	}
	applyConfig.AddApplyFlags(cmd.Flags())
	return cmd
}
