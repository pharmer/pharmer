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
			flags.EnsureRequiredFlags(cmd, "provider", "zone", "nodes", "kubernetes-version")

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
			CreateNodeGroups(ctx, cluster, nodes)
		},
	}

	cmd.Flags().StringVar(&cluster.Spec.Cloud.CloudProvider, "provider", "", "Provider name")
	cmd.Flags().StringVar(&cluster.Spec.Cloud.Zone, "zone", "", "Cloud provider zone name")
	cmd.Flags().StringVar(&cluster.Spec.CredentialName, "credential-uid", "", "Use preconfigured cloud credential uid")
	cmd.Flags().StringVar(&cluster.Spec.KubernetesVersion, "kubernetes-version", "", "Kubernetes version")
	cmd.Flags().StringVar(&cluster.Spec.KubeletVersion, "kubelet-version", "", "kubelet/kubectl version")
	cmd.Flags().StringVar(&cluster.Spec.KubeadmVersion, "kubeadm-version", "", "Kubeadm version")
	cmd.Flags().StringVar(&cluster.Spec.Networking.NetworkProvider, "network-provider", "calico", "Name of CNI plugin. Available options: calico, flannel, kubenet, weavenet")

	cmd.Flags().StringToIntVar(&nodes, "nodes", map[string]int{}, "Node set configuration")

	return cmd
}
