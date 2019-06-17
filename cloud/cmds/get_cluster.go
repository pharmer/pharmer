package cmds

import (
	"io"

	"github.com/appscode/go/term"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud/cmds/options"
	"github.com/pharmer/pharmer/store"
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

			err := RunGetCluster(opts, out)
			if err != nil {
				term.ExitOnError(err)
			}
		},
	}
	opts.AddFlags(cmd.Flags())

	return cmd
}

func RunGetCluster(opts *options.ClusterGetConfig, out io.Writer) error {

	rPrinter, err := printer.NewPrinter(opts.Output)
	if err != nil {
		return err
	}

	w := printer.GetNewTabWriter(out)

	clusters, err := getClusterList(opts.Clusters)
	if err != nil {
		return err
	}
	for _, cluster := range clusters {
		if err := rPrinter.PrintObj(cluster, w); err != nil {
			return err
		}
		err = printer.PrintNewline(w)
		if err != nil {
			return err
		}
	}

	w.Flush()
	return nil
}

func getClusterList(clusters []string) (clusterList []*api.Cluster, err error) {
	if len(clusters) != 0 {
		for _, arg := range clusters {
			cluster, er2 := store.StoreProvider.Clusters().Get(arg)
			if er2 != nil {
				return nil, er2
			}
			clusterList = append(clusterList, cluster)
		}

	} else {
		clusterList, err = store.StoreProvider.Clusters().List(metav1.ListOptions{})
		if err != nil {
			return
		}
	}
	return
}
