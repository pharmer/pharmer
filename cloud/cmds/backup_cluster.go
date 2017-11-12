package cmds

import (
	"context"
	"fmt"

	"github.com/appscode/go-term"
	"github.com/appscode/go/flags"
	"github.com/appscode/kutil/tools/backup"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/config"
	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type ItemList struct {
	Items []map[string]interface{} `json:"items,omitempty"`
}

func NewCmdBackup() *cobra.Command {
	var (
		clusterName string
		backupDir   string
		sanitize    bool
	)
	cmd := &cobra.Command{
		Use:               "cluster",
		Short:             "Backup cluster objects",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			flags.EnsureRequiredFlags(cmd, "cluster", "backup-dir")

			restConfig, err := searchLocalKubeConfig(clusterName)
			if err != nil || restConfig == nil {
				cfgFile, _ := config.GetConfigFile(cmd.Flags())
				cfg, err := config.LoadConfig(cfgFile)
				if err != nil {
					term.Fatalln(err)
				}
				ctx := cloud.NewContext(context.Background(), cfg, config.GetEnv(cmd.Flags()))

				cluster, err := cloud.Store(ctx).Clusters().Get(clusterName)
				if err != nil {
					term.Fatalln(err)
				}
				c2, err := cloud.GetAdminConfig(ctx, cluster)
				if err != nil {
					term.Fatalln(err)
				}
				restConfig = &rest.Config{
					Host: c2.Clusters[0].Cluster.Server,
					TLSClientConfig: rest.TLSClientConfig{
						CAData:   c2.Clusters[0].Cluster.CertificateAuthorityData,
						CertData: c2.AuthInfos[0].AuthInfo.ClientCertificateData,
						KeyData:  c2.AuthInfos[0].AuthInfo.ClientKeyData,
					},
				}
			}

			mgr := backup.NewBackupManager(clusterName, restConfig, sanitize)
			filename, err := mgr.BackupToTar(backupDir)
			if err != nil {
				term.Fatalln(err)
			}
			term.Successln(fmt.Sprintf("Cluster objects are stored in %s", filename))
		},
	}
	cmd.Flags().StringVarP(&clusterName, "cluster", "k", "", "Name of cluster")
	cmd.Flags().BoolVar(&sanitize, "sanitize", false, " Sanitize fields in YAML")
	cmd.Flags().StringVar(&backupDir, "backup-dir", "", "Directory where yaml files will be saved")
	return cmd
}

func searchLocalKubeConfig(clusterName string) (*rest.Config, error) {
	// ref: https://github.com/appscode/pharmer/blob/19db538fe51b83e807c525e2dbf28b56b7fb36e2/cloud/lib.go#L148
	ctxName := fmt.Sprintf("cluster-admin@%s.pharmer", clusterName)
	apiConfig, err := clientcmd.NewDefaultPathOptions().GetStartingConfig()
	if err != nil {
		return nil, err
	}
	overrides := &clientcmd.ConfigOverrides{CurrentContext: ctxName}
	return clientcmd.NewDefaultClientConfig(*apiConfig, overrides).ClientConfig()
}
