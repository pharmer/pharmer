package cmds

import (
	kubernetes "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/appctl/pkg/config"
	"github.com/appscode/appctl/pkg/util"
	term "github.com/appscode/go-term"
	"github.com/spf13/cobra"
)

func NewCmdDelete() *cobra.Command {
	var req kubernetes.ClusterDeleteRequest

	cmd := &cobra.Command{
		Use:               "delete",
		Short:             "Delete a Kubernetes cluster",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				req.Name = args[0]
				req.ReleaseReservedIp = true
			} else {
				term.Fatalln("Missing cluster name")
			}
			c := config.ClientOrDie()
			_, err := c.Kubernetes().V1beta1().Cluster().Delete(c.Context(), &req)
			util.PrintStatus(err)
			term.Successln("Request to delete cluster is accepted!")
		},
	}

	cmd.Flags().BoolVar(&req.Force, "force", false, "Force delete any running non-system apps")
	cmd.Flags().BoolVar(&req.ReleaseReservedIp, "release-reserved-ip", false, "Release reserved IP")
	cmd.Flags().BoolVar(&req.KeepLodabalancers, "keep-loadbalancers", false, "Keep loadbalancers")
	cmd.Flags().BoolVar(&req.DeleteDynamicVolumes, "delete-dynamic-volumes", false, "Delete dynamically provisioned volumes")

	return cmd
}
