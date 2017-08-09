package cmds

import (
	"bufio"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	api "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/appctl/pkg/config"
	"github.com/appscode/appctl/pkg/util"
	term "github.com/appscode/go-term"
	"github.com/appscode/go/flags"
	"github.com/spf13/cobra"
)

func NewCmdCreate() *cobra.Command {
	var req api.ClusterCreateRequest
	var nodes string

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a Kubernetes cluster for a given cloud provider",
		Example: "create --provider=(aws|gce|cc) --nodes=t1:n1,t2:n2 --zone=us-central1-f demo-cluster",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			flags.EnsureRequiredFlags(cmd, "provider", "zone", "nodes")

			if len(args) > 0 {
				req.Name = args[0]
			} else {
				term.Fatalln("missing cluster name")
			}

			if req.CloudCredential == "" {
				reader := bufio.NewReader(os.Stdin)
				data, err := ioutil.ReadAll(reader)
				term.ExitOnError(err)

				cred, err := util.ParseCloudCredential(string(data), req.Provider)
				term.ExitOnError(err)
				req.CloudCredentialData = cred
			}

			parts := strings.FieldsFunc(nodes, func(r rune) bool {
				return r == ',' || r == ':'
			})
			if len(parts)%2 != 0 {
				term.Fatalln("Nodes must be type:count pairs separated by ','")
			}
			req.NodeGroups = make([]*api.InstanceGroup, len(parts)/2)
			ng := 0
			for i := 0; i < len(parts)/2; i = i + 2 {
				sku := strings.TrimSpace(parts[i])
				count, err := strconv.ParseInt(strings.TrimSpace(parts[i+1]), 10, 64)
				if err != nil {
					term.Fatalln("Failed to parse count for nodes of type", sku)
				}
				req.NodeGroups[ng] = &api.InstanceGroup{
					Sku:   sku,
					Count: count,
				}
				ng++
			}

			c := config.ClientOrDie()
			_, err := c.Kubernetes().V1beta1().Cluster().Create(c.Context(), &req)
			util.PrintStatus(err)
			term.Successln("Request to create cluster is accepted!")
		},
	}

	cmd.Flags().StringVar(&req.Provider, "provider", "", "Provider name")
	cmd.Flags().StringVar(&req.Zone, "zone", "", "Cloud provider zone name")
	cmd.Flags().StringVar(&req.GceProject, "gce-project", "", "GCE project name(only applicable to `gce` provider)")
	cmd.Flags().StringVar(&nodes, "nodes", "", "Node set configuration")
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
