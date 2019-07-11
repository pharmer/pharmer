package cloud

import (
	"fmt"
	"io"

	"github.com/appscode/go/term"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cmds/cloud/options"
	"github.com/pharmer/pharmer/store"
	"github.com/pharmer/pharmer/utils/describer"
	"github.com/spf13/cobra"
	"k8s.io/kubernetes/pkg/kubectl/describe"
)

func NewCmdDescribeCluster(out io.Writer) *cobra.Command {
	opts := options.NewClusterDescribeConfig()
	cmd := &cobra.Command{
		Use: api.ResourceNameCluster,
		Aliases: []string{
			api.ResourceTypeCluster,
			api.ResourceKindCluster,
		},
		Short:             "Describe a Kubernetes cluster",
		Example:           "pharmer describe cluster <cluster_name>",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.ValidateFlags(cmd, args); err != nil {
				term.Fatalln(err)
			}

			storeProvider, err := store.GetStoreProvider(cmd)
			term.ExitOnError(err)

			err = RunDescribeCluster(storeProvider, opts, out)
			term.ExitOnError(err)
		},
	}

	return cmd
}

func RunDescribeCluster(storeProvider store.ResourceInterface, opts *options.ClusterDescribeConfig, out io.Writer) error {
	rDescriber := describer.NewDescriber()

	first := true
	clusters, err := getClusterList(storeProvider.Clusters(), opts.Clusters)
	if err != nil {
		return err
	}

	for _, cluster := range clusters {
		s, err := rDescriber.Describe(cluster, describe.DescriberSettings{})
		if err != nil {
			continue
		}
		if first {
			first = false
			fmt.Fprint(out, s)
		} else {
			fmt.Fprintf(out, "\n\n%s", s)
		}

		//if resp, err := cloud.CheckForUpdates(cluster.Name); err == nil {
		//	term.Println(resp)
		//} else {
		//	term.ExitOnError(err)
		//}
	}

	return nil
}