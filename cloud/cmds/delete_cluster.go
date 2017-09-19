package cmds

import (
	"context"

	"github.com/appscode/go-term"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/config"
	"github.com/spf13/cobra"
)

func NewCmdDeleteCluster() *cobra.Command {

	var (
		releaseReservedIP    = false
		force                = false
		keepLBs              = false
		deleteDynamicVolumes = false
	)

	cmd := &cobra.Command{
		Use: api.ResourceNameCluster,
		Aliases: []string{
			api.ResourceTypeCluster,
			api.ResourceKindCluster,
		},
		Short:             "Delete a Kubernetes cluster",
		Example:           "pharmer delete cluster demo-cluster",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				term.Fatalln("Missing cluster name.")
			}

			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			term.ExitOnError(err)

			ctx := cloud.NewContext(context.Background(), cfg)

			for _, clusterName := range args {
				_, err := cloud.Delete(ctx, clusterName)
				term.ExitOnError(err)
			}
		},
	}

	cmd.Flags().BoolVar(&force, "force", force, "Force delete any running non-system apps")
	cmd.Flags().BoolVar(&releaseReservedIP, "release-reserved-ip", releaseReservedIP, "Release reserved IP")
	cmd.Flags().BoolVar(&keepLBs, "keep-loadbalancers", keepLBs, "Keep loadbalancers")
	cmd.Flags().BoolVar(&deleteDynamicVolumes, "delete-dynamic-volumes", deleteDynamicVolumes, "Delete dynamically provisioned volumes")

	return cmd
}
