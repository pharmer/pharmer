package cmds

import (
	"context"
	"time"

	"github.com/appscode/go-term"
	"github.com/appscode/go/flags"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/config"
	"github.com/appscode/pharmer/phid"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewCmdCreateNodeGroup() *cobra.Command {

	nodes := map[string]int{}

	cmd := &cobra.Command{
		Use: api.ResourceNameNodeGroup,
		Aliases: []string{
			api.ResourceTypeNodeGroup,
			api.ResourceCodeNodeGroup,
			api.ResourceKindNodeGroup,
		},
		Short:             "Create a Kubernetes cluster NodeGroup for a given cloud provider",
		Example:           "pharmer create nodegroup -k <cluster_name>",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			flags.EnsureRequiredFlags(cmd, "cluster", "nodes")

			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			term.ExitOnError(err)

			ctx := cloud.NewContext(context.Background(), cfg)

			clusterName, _ := cmd.Flags().GetString("cluster")
			cluster, err := cloud.Get(ctx, clusterName)
			term.ExitOnError(err)

			for sku, count := range nodes {
				ig := api.NodeGroup{
					ObjectMeta: metav1.ObjectMeta{
						Name:              sku + "-pool",
						ClusterName:       cluster.Name,
						UID:               phid.NewNodeGroup(),
						CreationTimestamp: metav1.Time{Time: time.Now()},
						Labels: map[string]string{
							"node-role.kubernetes.io/node": "true",
						},
					},
					Spec: api.NodeGroupSpec{
						Nodes: int64(count),
						Template: api.NodeTemplateSpec{
							Spec: api.NodeSpec{
								SKU:           sku,
								SpotInstances: false,
								DiskType:      "pd-standard",
								DiskSize:      100,
							},
						},
					},
				}
				_, err := cloud.Store(ctx).NodeGroups(cluster.Name).Create(&ig)
				term.ExitOnError(err)
			}
		},
	}

	cmd.Flags().StringP("cluster", "k", "", "Name of the Kubernetes cluster")
	cmd.Flags().StringToIntVar(&nodes, "nodes", map[string]int{}, "Node set configuration")

	return cmd
}
