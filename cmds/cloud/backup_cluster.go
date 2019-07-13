package cloud

import (
	"fmt"

	"github.com/appscode/go/term"
	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"kmodules.xyz/client-go/tools/backup"
	"pharmer.dev/pharmer/cloud/utils/certificates"
	"pharmer.dev/pharmer/cloud/utils/kube"
	"pharmer.dev/pharmer/cmds/cloud/options"
	"pharmer.dev/pharmer/store"
)

type ItemList struct {
	Items []map[string]interface{} `json:"items,omitempty"`
}

func NewCmdBackup() *cobra.Command {
	opts := options.NewClusterBackupConfig()
	cmd := &cobra.Command{
		Use:               "cluster",
		Short:             "Backup cluster objects",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.ValidateFlags(cmd, args); err != nil {
				term.Fatalln(err)
			}

			storeProvider, err := store.GetStoreProvider(cmd)
			term.ExitOnError(err)

			restConfig, err := SearchLocalKubeConfig(opts.ClusterName)
			if err != nil || restConfig == nil {
				cluster, err := storeProvider.Clusters().Get(opts.ClusterName)
				if err != nil {
					term.Fatalln(err)
				}

				caCert, caKey, err := storeProvider.Certificates(cluster.Name).Get("ca")
				term.ExitOnError(err)

				c2, err := kube.GetAdminConfig(cluster, &certificates.CertKeyPair{Cert: caCert, Key: caKey})

				if err != nil {
					term.Fatalln(err)
				}
				restConfig = kube.NewRestConfigFromKubeConfig(c2)
			}

			mgr := backup.NewBackupManager(opts.ClusterName, restConfig, opts.Sanitize)
			filename, err := mgr.BackupToTar(opts.BackupDir)
			if err != nil {
				term.Fatalln(err)
			}
			term.Successln(fmt.Sprintf("Cluster objects are stored in %s", filename))
		},
	}
	opts.AddFlags(cmd.Flags())
	return cmd
}

func SearchLocalKubeConfig(clusterName string) (*rest.Config, error) {
	// ref: https://pharmer.dev/pharmer/blob/19db538fe51b83e807c525e2dbf28b56b7fb36e2/cloud/lib.go#L148
	ctxName := fmt.Sprintf("cluster-admin@%s.pharmer", clusterName)
	apiConfig, err := clientcmd.NewDefaultPathOptions().GetStartingConfig()
	if err != nil {
		return nil, err
	}
	overrides := &clientcmd.ConfigOverrides{CurrentContext: ctxName}
	return clientcmd.NewDefaultClientConfig(*apiConfig, overrides).ClientConfig()
}
