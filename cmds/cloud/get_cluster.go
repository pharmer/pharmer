package cloud

import (
	"io"

	"github.com/appscode/go/term"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	api "pharmer.dev/pharmer/apis/v1alpha1"
	"pharmer.dev/pharmer/cmds/cloud/options"
	"pharmer.dev/pharmer/store"
	"pharmer.dev/pharmer/utils/printer"
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

			storeProvider, err := store.GetStoreProvider(cmd)
			if err != nil {
				term.Fatalln(err)
			}

			err = runGetCluster(storeProvider.Clusters(), opts, out)
			if err != nil {
				term.ExitOnError(err)
			}
		},
	}
	opts.AddFlags(cmd.Flags())

	return cmd
}

func runGetCluster(clusterStore store.ClusterStore, opts *options.ClusterGetConfig, out io.Writer) error {
	rPrinter, err := printer.NewPrinter(opts.Output)
	if err != nil {
		return err
	}

	w := printer.GetNewTabWriter(out)

	clusters, err := getClusterList(clusterStore, opts.Clusters)
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

	return w.Flush()
}

func getClusterList(clusterStore store.ClusterStore, clusters []string) (clusterList []*api.Cluster, err error) {
	if len(clusters) != 0 {
		for _, arg := range clusters {
			cluster, er2 := clusterStore.Get(arg)
			if er2 != nil {
				return nil, er2
			}
			clusterList = append(clusterList, cluster)
		}

	} else {
		clusterList, err = clusterStore.List(metav1.ListOptions{})
		if err != nil {
			return
		}
	}
	return
}
