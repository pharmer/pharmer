package cmds

import (
	"os"
	"strconv"

	api "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/appctl/pkg/config"
	"github.com/appscode/appctl/pkg/util"
	"github.com/appscode/appctl/pkg/util/timeutil"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

func NewCmdList() *cobra.Command {
	var req api.ClusterListRequest

	cmd := &cobra.Command{
		Use:               "list",
		Short:             "Lists active Kubernetes clusters",
		Example:           "appctl cluster list",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			c := config.ClientOrDie()
			resp, err := c.Kubernetes().V1beta1().Cluster().List(c.Context(), &req)
			util.PrintStatus(err)

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Name", "Provider", "Zone", "Api Server URL", "Number of Nodes", "Version", "Running Since"})

			for _, cluster := range resp.Clusters {
				version := cluster.Version
				if version == "" {
					version = cluster.KubeletVersion
				}
				table.Append([]string{cluster.Name,
					cluster.Provider,
					cluster.Zone,
					cluster.ApiServerUrl,
					strconv.Itoa(int(cluster.NodeCount)),
					version,
					timeutil.Format(cluster.CreatedAt),
				})
			}
			table.Render()
		},
	}

	return cmd
}
