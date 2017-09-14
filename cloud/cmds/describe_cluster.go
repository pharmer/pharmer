package cmds

import (
	"context"
	"io"

	"github.com/appscode/log"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/config"
	"github.com/spf13/cobra"
	"github.com/appscode/pharmer/cloud/describer"
	"fmt"
	"k8s.io/kubernetes/pkg/printers"
)

func NewCmdDescribeCluster(out, errOut io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use: api.ResourceNameCluster,
		Aliases: []string{
			api.ResourceTypeCluster,
			api.ResourceCodeCluster,
			api.ResourceKindCluster,
		},
		Short:             "Describe a Kubernetes cluster",
		Example:           "pharmer describe cluster <cluster_name>",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				log.Fatalln(err)
			}
			ctx := cloud.NewContext(context.Background(), cfg)

			RunDescribeCluster(ctx, cmd, out, errOut, args)

		},
	}

	return cmd
}

func RunDescribeCluster(ctx context.Context, cmd *cobra.Command, out, errOut io.Writer, args []string) error {

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
	}

	return nil
}
