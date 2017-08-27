package cmds

import (
	"errors"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/spf13/cobra"
)

func NewCmdReconfigure() *cobra.Command {
	var req proto.ClusterReconfigureRequest

	cmd := &cobra.Command{
		Use:               "reconfigure",
		Short:             "Create/Resize/Upgrade/Downgrade a Kubernetes cluster instance group",
		Example:           `appctl cluster reconfigure <name> --role=master|node --sku=n1-standard-1`,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				req.Name = args[0]
			} else {
				return errors.New("missing cluster name")
			}

			return reconfigure(&req)
		},
	}

	cmd.Flags().BoolVar(&req.ApplyToMaster, "apply-to-master", false, "Set true to change version of master. Default set to false.")
	cmd.Flags().StringVar(&req.Sku, "sku", "", "Instance type")
	cmd.Flags().Int64Var(&req.Count, "count", -1, "Number of instances of this type")
	cmd.Flags().StringVar(&req.KubernetesVersion, "kubernetes-version", "", "Kubernetes server version")

	return cmd
}

func reconfigure(req *proto.ClusterReconfigureRequest) error {
	return nil
}
