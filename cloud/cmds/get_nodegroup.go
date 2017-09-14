package cmds

import (
	"context"
	"io"

	"github.com/appscode/log"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/cloud/printer"
	"github.com/appscode/pharmer/config"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewCmdGetNodeGroup(out, errOut io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use: api.ResourceNameNodeGroup,
		Aliases: []string{
			api.ResourceTypeNodeGroup,
			api.ResourceCodeNodeGroup,
			api.ResourceKindNodeGroup,
		},
		Short:             "Get a Kubernetes cluster NodeGroup",
		Example:           "pharmer get nodegroup -k <cluster_name>",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				log.Fatalln(err)
			}
			ctx := cloud.NewContext(context.Background(), cfg)

			RunGetNodeGroup(ctx, cmd, out, errOut, args)

		},
	}

	cmd.Flags().StringP("cluster", "k", "", "Name of the Kubernetes cluster")
	cmd.Flags().StringP("output", "o", "", "Output format. One of: json|yaml|wide")
	return cmd
}

func RunGetNodeGroup(ctx context.Context, cmd *cobra.Command, out, errOut io.Writer, args []string) error {

	rPrinter, err := printer.NewPrinter(cmd)
	if err != nil {
		return err
	}

	w := printer.GetNewTabWriter(out)

	clusterList := make([]string, 0)
	clusterName, _ := cmd.Flags().GetString("cluster")

	if clusterName != "" {
		clusterList = append(clusterList, clusterName)
	} else {
		clusters, err := cloud.Store(ctx).Clusters().List(metav1.ListOptions{})
		if err != nil {
			return err
		}
		for _, c := range clusters {
			clusterList = append(clusterList, c.Name)
		}
	}

	for _, cluster := range clusterList {
		nodegroups, err := getNodeGroupList(ctx, cluster, args...)
		if err != nil {
			return err
		}
		if len(nodegroups) == 0 {
			continue
		}

		for _, ng := range nodegroups {
			if err := rPrinter.PrintObj(ng, w); err != nil {
				return err
			}
			if rPrinter.IsGeneric() {
				printer.PrintNewline(w)
			}
		}

		rPrinter.(*printer.HumanReadablePrinter).PrintHeader(true)
	}

	w.Flush()
	return nil
}

func getNodeGroupList(ctx context.Context, cluster string, args ...string) (nodeGroupList []*api.NodeGroup, err error) {
	if len(args) != 0 {
		for _, arg := range args {
			nodeGroup, er2 := cloud.Store(ctx).NodeGroups(cluster).Get(arg)
			if er2 != nil {
				return nil, er2
			}
			nodeGroupList = append(nodeGroupList, nodeGroup)
		}

	} else {
		nodeGroupList, err = cloud.Store(ctx).NodeGroups(cluster).List(metav1.ListOptions{})
		if err != nil {
			return
		}
	}
	return
}
