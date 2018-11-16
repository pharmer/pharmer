package cmds

import (
	"context"
	"io"

	"github.com/appscode/go/term"
	api "github.com/pharmer/pharmer/apis/v1alpha1"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/cloud/cmds/options"
	"github.com/pharmer/pharmer/config"
	"github.com/pharmer/pharmer/utils/printer"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewCmdGetCluster(out io.Writer) *cobra.Command {
	opts := options.NewClusterGetConfig()
	cmd := &cobra.Command{
		Use: api.ResourceNameCluster,
		Aliases: []string{
			api.ResourceTypeCluster,
			api.ResourceKindCluster,
		},
		Short:             "Get a Kubernetes cluster",
		Example:           "pharmer get cluster",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.ValidateFlags(cmd, args); err != nil {
				term.Fatalln(err)
			}
			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			term.ExitOnError(err)

			ctx := cloud.NewContext(context.Background(), cfg, config.GetEnv(cmd.Flags()))
			RunGetCluster(ctx, opts, out)
		},
	}
	opts.AddFlags(cmd.Flags())

	return cmd
}

func RunGetCluster(ctx context.Context, opts *options.ClusterGetConfig, out io.Writer) error {

	rPrinter, err := printer.NewPrinter(opts.Output)
	if err != nil {
		return err
	}

	w := printer.GetNewTabWriter(out)

	clusters, err := getClusterList(ctx, opts.Clusters, opts.Owner)
	if err != nil {
		return err
	}
	for _, cluster := range clusters {
		if err := rPrinter.PrintObj(cluster, w); err != nil {
			return err
		}
		printer.PrintNewline(w)
	}

	w.Flush()
	return nil
}

func getClusterList(ctx context.Context, clusters []string, owner string) (clusterList []*api.Cluster, err error) {
	if len(clusters) != 0 {
		for _, arg := range clusters {
			cluster, er2 := cloud.Store(ctx).Owner(owner).Clusters().Get(arg)
			if er2 != nil {
				return nil, er2
			}
			clusterList = append(clusterList, cluster)
		}

	} else {
		clusterList, err = cloud.Store(ctx).Owner(owner).Clusters().List(metav1.ListOptions{})
		if err != nil {
			return
		}
	}
	return
}
