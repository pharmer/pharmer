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

			storeProvider, err := store.GetStoreProvider(cmd, opts.Owner)
			if err != nil {
				term.Fatalln(err)
			}

			err = runGetCluster(storeProvider, opts, out)
			if err != nil {
				term.ExitOnError(err)
			}
		},
	}
	opts.AddFlags(cmd.Flags())

	return cmd
}

func runGetCluster(storeProvider store.ResourceInterface, opts *options.ClusterGetConfig, out io.Writer) error {

	rPrinter, err := printer.NewPrinter(opts.Output)
	if err != nil {
		return err
	}

	w := printer.GetNewTabWriter(out)

	clusters, err := getClusterList(storeProvider, opts.Clusters)
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

func getClusterList(storeProvider store.ResourceInterface, clusters []string) (clusterList []*api.Cluster, err error) {
	if len(clusters) != 0 {
		for _, arg := range clusters {
			cluster, er2 := storeProvider.Clusters().Get(arg)
			if er2 != nil {
				return nil, er2
			}
			clusterList = append(clusterList, cluster)
		}

	} else {
		clusterList, err = storeProvider.Clusters().List(metav1.ListOptions{})
		if err != nil {
			return
		}
	}
	return
}
