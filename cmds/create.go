package cmds

import (
	"bufio"
	"errors"
	"io/ioutil"
	"os"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/go/flags"
	"github.com/appscode/pharmer/credential"
	"github.com/spf13/cobra"
)

func NewCmdCreate() *cobra.Command {
	var req proto.ClusterCreateRequest
	nodes := map[string]int{}

	cmd := &cobra.Command{
		Use:               "create",
		Short:             "Create a Kubernetes cluster for a given cloud provider",
		Example:           "create --provider=(aws|gce|cc) --nodes=t1=1,t2=2 --zone=us-central1-f demo-cluster",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.EnsureRequiredFlags(cmd, "provider", "zone", "nodes")

			if len(args) > 0 {
				req.Name = args[0]
			} else {
				return errors.New("missing cluster name")
			}

			if req.CloudCredential == "" {
				reader := bufio.NewReader(os.Stdin)
				data, err := ioutil.ReadAll(reader)
				if err != nil {
					return err
				}

				cred, err := credential.ParseCloudCredential(string(data), req.Provider)
				if err != nil {
					return err
				}
				req.CloudCredentialData = cred
			}

			req.NodeGroups = make([]*proto.InstanceGroup, len(nodes))
			ng := 0
			for sku, count := range nodes {
				req.NodeGroups[ng] = &proto.InstanceGroup{
					Sku:   sku,
					Count: int64(count),
				}
				ng++
			}
			return create(&req)
		},
	}

	cmd.Flags().StringVar(&req.Provider, "provider", "", "Provider name")
	cmd.Flags().StringVar(&req.Zone, "zone", "", "Cloud provider zone name")
	cmd.Flags().StringVar(&req.GceProject, "gce-project", "", "GCE project name(only applicable to `gce` provider)")
	cmd.Flags().StringToIntVar(&nodes, "nodes", map[string]int{}, "Node set configuration")
	cmd.Flags().StringVar(&req.CloudCredential, "cloud-credential", "", "Use preconfigured cloud credential phid")
	cmd.Flags().StringVar(&req.SaltbaseVersion, "saltbase-version", "", "Kubernetes saltbase version")
	cmd.Flags().StringVar(&req.KubeStarterVersion, "kube-starter-version", "", "Kube starter version")
	cmd.Flags().StringVar(&req.KubeletVersion, "kubelet-version", "", "Kubernetes server version")
	cmd.Flags().StringVar(&req.HostfactsVersion, "hostfacts-version", "", "Hostfacts version")
	cmd.Flags().StringVar(&req.Version, "version", "", "Kubernetes version")
	cmd.Flags().BoolVar(&req.DoNotDelete, "do-not-delete", false, "Set do not delete flag")

	cmd.Flags().MarkHidden("saltbase-version")
	cmd.Flags().MarkHidden("kube-starter-version")
	cmd.Flags().MarkHidden("kubelet-version")
	cmd.Flags().MarkHidden("hostfacts-version")

	return cmd
}

func create(req *proto.ClusterCreateRequest) error {
	return nil
}
