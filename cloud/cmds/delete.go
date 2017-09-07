package cmds

import (
	"github.com/spf13/cobra"
)

func NewCmdDelete() *cobra.Command {
	var (
		releaseReservedIP    = false
		force                = false
		keepLBs              = false
		deleteDynamicVolumes = false
	)

	cmd := &cobra.Command{
		Use:               "delete",
		Short:             "Delete a Kubernetes cluster",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
		},
	}

	cmd.Flags().BoolVar(&force, "force", force, "Force delete any running non-system apps")
	cmd.Flags().BoolVar(&releaseReservedIP, "release-reserved-ip", releaseReservedIP, "Release reserved IP")
	cmd.Flags().BoolVar(&keepLBs, "keep-loadbalancers", keepLBs, "Keep loadbalancers")
	cmd.Flags().BoolVar(&deleteDynamicVolumes, "delete-dynamic-volumes", deleteDynamicVolumes, "Delete dynamically provisioned volumes")

	return cmd
}
