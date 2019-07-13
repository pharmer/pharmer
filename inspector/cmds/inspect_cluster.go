package cmds

import (
	"github.com/appscode/go/flags"
	"github.com/appscode/go/term"
	"github.com/spf13/cobra"
	api "pharmer.dev/pharmer/apis/v1beta1"
	"pharmer.dev/pharmer/cloud/utils/certificates"
	"pharmer.dev/pharmer/inspector"
	"pharmer.dev/pharmer/store"
)

func NewCmdInspectCluster() *cobra.Command {
	cmd := &cobra.Command{
		Use: api.ResourceNameCluster,
		Aliases: []string{
			api.ResourceTypeCluster,
			api.ResourceKindCluster,
		},
		Short:             "Inspect cluster object",
		Example:           `pharmer inspect cluster -k <cluster-name>  <inspect-type>`,
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			flags.EnsureRequiredFlags(cmd, "kubernetes-version")

			if len(args) > 1 {
				term.Fatalln("Multiple inspect type provided.")
			}

			clusterName, _ := cmd.Flags().GetString("cluster")

			storeProvider, err := store.GetStoreProvider(cmd)
			term.ExitOnError(err)

			cluster, err := storeProvider.Clusters().Get(clusterName)
			term.ExitOnError(err)

			caCert, caKey, err := storeProvider.Certificates(clusterName).Get("ca")
			term.ExitOnError(err)

			inspect, err := inspector.New(storeProvider, cluster, &certificates.CertKeyPair{
				Cert: caCert,
				Key:  caKey,
			})
			if err != nil {
				term.Fatalln(err)
			}
			if len(args) == 0 {
				if err = inspect.NetworkCheck(); err != nil {
					term.Fatalln(err)
				}
				if err = inspect.NativeCheck(); err != nil {
					term.Fatalln(err)
				}
			} else {
				switch args[0] {
				case "network":
					if err = inspect.NetworkCheck(); err != nil {
						term.Fatalln(err)
					}
				case "native":
					if err = inspect.NativeCheck(); err != nil {
						term.Fatalln(err)
					}
				default:
					term.Fatalln("Unknown inspect type")

				}
			}

		},
	}

	cmd.Flags().StringP("cluster", "k", "", "Name of the Kubernetes cluster")

	return cmd
}
