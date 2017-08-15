package cmds

import (
	"errors"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/spf13/cobra"
)

func NewCmdDelete() *cobra.Command {
	var req proto.ClusterDeleteRequest

	cmd := &cobra.Command{
		Use:               "delete",
		Short:             "Delete a Kubernetes cluster",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				req.Name = args[0]
				req.ReleaseReservedIp = true
			} else {
				return errors.New("Missing cluster name")
			}

			return delete2(&req)
		},
	}

	cmd.Flags().BoolVar(&req.Force, "force", false, "Force delete any running non-system apps")
	cmd.Flags().BoolVar(&req.ReleaseReservedIp, "release-reserved-ip", false, "Release reserved IP")
	cmd.Flags().BoolVar(&req.KeepLodabalancers, "keep-loadbalancers", false, "Keep loadbalancers")
	cmd.Flags().BoolVar(&req.DeleteDynamicVolumes, "delete-dynamic-volumes", false, "Delete dynamically provisioned volumes")

	return cmd
}

func delete2(req *proto.ClusterDeleteRequest) error {
	return nil
}
