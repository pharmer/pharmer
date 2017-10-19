package cmds

import (
	"context"
	"io"

	"github.com/appscode/go-term"
	api "github.com/appscode/pharmer/apis/v1alpha1"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/cloud/printer"
	"github.com/appscode/pharmer/config"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewCmdGetCluster(out io.Writer) *cobra.Command {
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
			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			term.ExitOnError(err)

			ctx := cloud.NewContext(context.Background(), cfg, config.GetEnv(cmd.Flags()))
			RunGetCluster(ctx, cmd, out, args)
		},
	}

	cmd.Flags().StringP("output", "o", "", "Output format. One of: json|yaml|wide")
	return cmd
}

func RunGetCluster(ctx context.Context, cmd *cobra.Command, out io.Writer, args []string) error {

	rPrinter, err := printer.NewPrinter(cmd)
	if err != nil {
		return err
	}

	w := printer.GetNewTabWriter(out)

	clusters, err := getClusterList(ctx, args)
	if err != nil {
		return err
	}
	for _, cluster := range clusters {
		if err := rPrinter.PrintObj(cluster, w); err != nil {
			return err
		}
		if rPrinter.IsGeneric() {
			printer.PrintNewline(w)
		}
	}

	w.Flush()
	return nil
}

func getClusterList(ctx context.Context, args []string) (clusterList []*api.Cluster, err error) {
	if len(args) != 0 {
		for _, arg := range args {
			cluster, er2 := cloud.Store(ctx).Clusters().Get(arg)
			if er2 != nil {
				return nil, er2
			}
			clusterList = append(clusterList, cluster)
		}

	} else {
		clusterList, err = cloud.Store(ctx).Clusters().List(metav1.ListOptions{})
		if err != nil {
			return
		}
	}
	return
}
