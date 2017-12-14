package cmds

import (
	"context"
	"fmt"
	"io"

	"github.com/appscode/go/term"
	api "github.com/pharmer/pharmer/apis/v1alpha1"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/cloud/cmds/options"
	"github.com/pharmer/pharmer/config"
	"github.com/pharmer/pharmer/utils/describer"
	"github.com/spf13/cobra"
	"k8s.io/kubernetes/pkg/printers"
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
			if err := opts.ValidateClusterDescribeFlags(cmd, args); err != nil {
				term.Fatalln(err)
			}
			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			term.ExitOnError(err)

			ctx := cloud.NewContext(context.Background(), cfg, config.GetEnv(cmd.Flags()))
			err = RunDescribeCluster(ctx, out, opts.Clusters)
			term.ExitOnError(err)
		},
	}
	opts.AddClusterDescribeFlags(cmd.Flags())

	return cmd
}

func RunDescribeCluster(ctx context.Context, out io.Writer, args []string) error {
	rDescriber := describer.NewDescriber(ctx)

	first := true
	clusters, err := getClusterList(ctx, args)
	if err != nil {
		return err
	}

	for _, cluster := range clusters {
		s, err := rDescriber.Describe(cluster, &printers.DescriberSettings{})
		if err != nil {
			continue
		}
		if first {
			first = false
			fmt.Fprint(out, s)
		} else {
			fmt.Fprintf(out, "\n\n%s", s)
		}

		if resp, err := cloud.CheckForUpdates(ctx, cluster.Name); err == nil {
			term.Println(resp)
		}
	}

	return nil
}
