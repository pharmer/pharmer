package cmds

import (
	"context"

	"github.com/appscode/go/flags"
	"github.com/appscode/go/log"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/config"
	"github.com/spf13/cobra"
)

func NewCmdDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "delete",
		Short:             "Delete a Kubernetes clusters, nodegroups",
		DisableAutoGenTag: true,
	}
	cmd.AddCommand(newCmdClusterDelete())
	cmd.AddCommand(newCmdIgDelete())
	return cmd
}

func newCmdClusterDelete() *cobra.Command {
	var (
		releaseReservedIP    = false
		force                = false
		keepLBs              = false
		deleteDynamicVolumes = false
	)
	cluster := &api.Cluster{}

	cmd := &cobra.Command{
		Use:               "cluster",
		Short:             "Delete a Kubernetes cluster",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			//flags.EnsureRequiredFlags(cmd, "provider", "zone", "nodes")

			if len(args) == 0 {
				log.Fatalln("Missing cluster name")
			}
			if len(args) > 1 {
				log.Fatalln("Multiple cluster name provided.")
			}
			cluster.Name = args[0]
			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				log.Fatalln(err)
			}
			ctx := cloud.NewContext(context.Background(), cfg)
			cluster, err = cloud.Delete(ctx, cluster.Name)
			if err != nil {
				log.Fatalln(err)
			}
		},
	}
	cmd.Flags().BoolVar(&force, "force", force, "Force delete any running non-system apps")
	cmd.Flags().BoolVar(&releaseReservedIP, "release-reserved-ip", releaseReservedIP, "Release reserved IP")
	cmd.Flags().BoolVar(&keepLBs, "keep-loadbalancers", keepLBs, "Keep loadbalancers")
	cmd.Flags().BoolVar(&deleteDynamicVolumes, "delete-dynamic-volumes", deleteDynamicVolumes, "Delete dynamically provisioned volumes")
	return cmd
}

func newCmdIgDelete() *cobra.Command {
	var (
		sku string
	)
	cluster := &api.Cluster{}

	cmd := &cobra.Command{
		Use:               "ng",
		Short:             "Delete a Kubernetes nodegroups",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			flags.EnsureRequiredFlags(cmd, "nodes")

			if len(args) == 0 {
				log.Fatalln("Missing cluster name")
			}
			if len(args) > 1 {
				log.Fatalln("Multiple cluster name provided.")
			}
			cluster.Name = args[0]
			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				log.Fatalln(err)
			}
			ctx := cloud.NewContext(context.Background(), cfg)
			cluster, err = cloud.DeleteNG(ctx, sku, cluster.Name)
			if err != nil {
				log.Fatalln(err)
			}
		},
	}
	cmd.Flags().StringVar(&sku, "sku", "", "sku of a nodegroup")
	return cmd
}
