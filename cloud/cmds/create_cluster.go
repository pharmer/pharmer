package cmds

import (
	"context"

	"github.com/appscode/go/flags"
	"github.com/appscode/go/term"
	api "github.com/appscode/pharmer/apis/v1alpha1"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/config"
	"github.com/spf13/cobra"
)

func NewCmdCreateCluster() *cobra.Command {
	var spotInstance bool
	var spotPriceMax float64
	cluster := &api.Cluster{}
	nodes := map[string]int{}

	cmd := &cobra.Command{
		Use: api.ResourceNameCluster,
		Aliases: []string{
			api.ResourceTypeCluster,
			api.ResourceKindCluster,
		},
		Short:             "Create a Kubernetes cluster for a given cloud provider",
		Example:           "pharmer create cluster demo-cluster",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			spot_price_max := ""
			if spotInstance {
				spot_price_max = "spot-price-max"
			}
			flags.EnsureRequiredFlags(cmd, "provider", "zone", "kubernetes-version", spot_price_max)

			if len(args) == 0 {
				term.Fatalln("Missing cluster name.")
			}
			if len(args) > 1 {
				term.Fatalln("Multiple cluster name provided.")
			}

			cluster.Name = args[0]
			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				term.Fatalln(err)
			}
			ctx := cloud.NewContext(context.Background(), cfg, config.GetEnv(cmd.Flags()))
			cluster, err = cloud.Create(ctx, cluster)
			if err != nil {
				term.Fatalln(err)
			}
			if len(nodes) > 0 {
				CreateNodeGroups(ctx, cluster, nodes, spotInstance, spotPriceMax)
			}
		},
	}

	cmd.Flags().StringVar(&cluster.Spec.Cloud.CloudProvider, "provider", "", "Provider name")
	cmd.Flags().StringVar(&cluster.Spec.Cloud.Zone, "zone", "", "Cloud provider zone name")
	cmd.Flags().StringVar(&cluster.Spec.CredentialName, "credential-uid", "", "Use preconfigured cloud credential uid")
	cmd.Flags().StringVar(&cluster.Spec.KubernetesVersion, "kubernetes-version", "", "Kubernetes version")
	cmd.Flags().StringVar(&cluster.Spec.KubeletVersion, "kubelet-version", "", "kubelet/kubectl version")
	cmd.Flags().StringVar(&cluster.Spec.KubeadmVersion, "kubeadm-version", "", "Kubeadm version")
	cmd.Flags().StringVar(&cluster.Spec.Networking.NetworkProvider, "network-provider", "calico", "Name of CNI plugin. Available options: calico, flannel, kubenet, weavenet")
	cmd.Flags().BoolVar(&spotInstance, "spot-instance", false, "Set spot instance flag")
	cmd.Flags().Float64Var(&spotPriceMax, "spot-price-max", float64(0), "Maximum price of spot instance")

	cmd.Flags().StringToIntVar(&nodes, "nodes", map[string]int{}, "Node set configuration")

	return cmd
}
