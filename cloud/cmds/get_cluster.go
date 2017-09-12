package cmds

import (
	"context"
	"io"

	"github.com/appscode/appctl/pkg/util/timeutil"
	"github.com/appscode/log"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/config"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewCmdGetCluster(out io.Writer, errOut io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "cluster",
		Short:             "Get list of active Kubernetes clusters",
		Example:           "pharmer get cluster <name>",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				log.Fatalln(err)
			}
			ctx := cloud.NewContext(context.Background(), cfg)

			RunGet(ctx, cmd, out, errOut, args)

			return nil
		},
	}

	return cmd
}

func RunGet(ctx context.Context, cmd *cobra.Command, out, errOut io.Writer, args []string) error {
	clusterList, err := getClusterList(ctx, args)
	if err != nil {
		return err
	}

	//rPrinter, err := printer.NewPrinter(cmd)
	//if err != nil {
	//	return err
	//}
	//
	//w := printer.GetNewTabWriter(out)

	table := tablewriter.NewWriter(out)
	table.SetHeader([]string{"Name", "Provider", "Zone", "Version", "Running Since"})

	for _, cluster := range clusterList {
		//if resourcePrinter, found := rPrinter.(*printer.HumanReadablePrinter); found {
		//	if err := rPrinter.PrintObj(cluster, w); err != nil {
		//	}
		//	continue
		//}
		//
		//if err := rPrinter.PrintObj(cluster, w); err != nil {
		//	continue
		//}

		table.Append([]string{cluster.Name,
			cluster.Spec.Cloud.CloudProvider,
			cluster.Spec.Cloud.Zone,
			//cluster.ApiServerUrl,
			//strconv.Itoa(int(cluster.NodeCount)),
			cluster.Spec.KubernetesVersion,
			timeutil.Format(cluster.CreationTimestamp.Unix()),
		})
	}
	table.Render()
	return nil
}

func getClusterList(ctx context.Context, args []string) (clusterList []*api.Cluster, err error) {

	if len(args) == 1 {
		cluster, er2 := cloud.Store(ctx).Clusters().Get(args[0])
		if er2 != nil {
			return nil, er2
		}
		clusterList = append(clusterList, cluster)
	} else {
		clusterList, err = cloud.Store(ctx).Clusters().List(metav1.ListOptions{})
		if err != nil {
			return
		}
	}
	return
}
