package cmds

import (
	"context"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/go/log"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/config"
	"github.com/spf13/cobra"
)

func NewCmdApply() *cobra.Command {
	var (
		dryRun bool
	)
	cmd := &cobra.Command{
		Use:               "apply",
		Short:             "Apply changes",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				log.Fatalln("Missing cluster name")
			}
			if len(args) > 1 {
				log.Fatalln("Multiple cluster name provided.")
			}
			name := args[0]

			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				log.Fatalln(err)
			}
			ctx := cloud.NewContext(context.Background(), cfg)

			err = cloud.Apply(ctx, name, dryRun)
			if err != nil {
				log.Fatalln(err)
			}
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", dryRun, "Dry run.")
	return cmd
}

func reconfigure(req *proto.ClusterReconfigureRequest) error {
	return nil
}
