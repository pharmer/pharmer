package cmds

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/appscode/go/ioutil"
	"github.com/appscode/go/log"
	"github.com/appscode/go/term"
	api "github.com/pharmer/pharmer/apis/v1alpha1"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/cloud/cmds/options"
	"github.com/pharmer/pharmer/config"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"
)

func NewCmdUse() *cobra.Command {
	opts := options.NewClusterUseConfig()
	cmd := &cobra.Command{
		Use:               "cluster",
		Short:             "Sets `kubectl` context to given cluster",
		Example:           `pharmer use cluster <name>`,
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.ValidateFlags(cmd, args); err != nil {
				term.Fatalln(err)
			}
			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				log.Fatalln(err)
			}
			ctx := cloud.NewContext(context.Background(), cfg, config.GetEnv(cmd.Flags()))

			cluster, err := cloud.Store(ctx).Clusters().Get(opts.ClusterName)
			if err != nil {
				term.Fatalln(err)
			}
			c2, err := cloud.GetAdminConfig(ctx, cluster)
			if err != nil {
				log.Fatalln(err)
			}
			UseCluster(ctx, opts, c2)
		},
	}
	opts.AddFlags(cmd.Flags())
	return cmd
}

func UseCluster(ctx context.Context, opts *options.ClusterUseConfig, konf *api.KubeConfig) {
	var konfig *clientcmdapi.Config
	if _, err := os.Stat(KubeConfigPath()); err == nil {
		// ~/.kube/config exists
		konfig, err = clientcmd.LoadFromFile(KubeConfigPath())
		if err != nil {
			log.Fatalln(err)
		}

		bakFile := KubeConfigPath() + ".bak." + time.Now().Format("2006-01-02T15-04")
		err = ioutil.CopyFile(bakFile, KubeConfigPath(), 0600)
		if err != nil {
			log.Fatalln(err)
		}
		term.Infoln(fmt.Sprintf("Current Kubeconfig is backed up as %s.", bakFile))
	} else {
		konfig = &clientcmdapi.Config{
			APIVersion: "v1",
			Kind:       "Config",
			Preferences: clientcmdapi.Preferences{
				Colors: true,
			},
		}
	}

	ctxName := fmt.Sprintf("cluster-admin@%s.pharmer", opts.ClusterName)

	if !opts.Overwrite {
		if konfig.CurrentContext == ctxName {
			term.Infoln(fmt.Sprintf("Cluster `%s` is already current context.", opts.ClusterName))
			os.Exit(0)
		}
	}

	_, found := konfig.Contexts[ctxName]
	if !found || opts.Overwrite {
		// Upsert cluster
		if c, found := konfig.Clusters[konf.Cluster.Name]; found {
			setCluster(c, konf.Cluster)
		} else {
			konfig.Clusters[konf.Cluster.Name] = setCluster(&clientcmdapi.Cluster{}, konf.Cluster)
		}

		// Upsert user
		if au, found := konfig.AuthInfos[konf.AuthInfo.Name]; found {
			setUser(au, konf.AuthInfo)
		} else {
			konfig.AuthInfos[konf.AuthInfo.Name] = setUser(&clientcmdapi.AuthInfo{}, konf.AuthInfo)
		}

		// Upsert context
		if c, found := konfig.Contexts[konf.Context.Name]; found {
			setContext(c, konf.Context)
		} else {
			konfig.Contexts[konf.Context.Name] = setContext(&clientcmdapi.Context{}, konf.Context)
		}
	}

	// Update current context
	konfig.CurrentContext = ctxName

	err := os.MkdirAll(filepath.Dir(KubeConfigPath()), 0755)
	if err != nil {
		log.Fatalln(err)
	}
	err = clientcmd.WriteToFile(*konfig, KubeConfigPath())
	if err != nil {
		log.Fatalln(err)
	}
	term.Successln(fmt.Sprintf("kubectl context set to cluster `%s`.", opts.ClusterName))
}

func setCluster(cur *clientcmdapi.Cluster, desired api.NamedCluster) *clientcmdapi.Cluster {
	*cur = clientcmdapi.Cluster{
		Server: desired.Server,
		CertificateAuthorityData: append([]byte(nil), desired.CertificateAuthorityData...),
	}
	return cur
}

func setUser(cur *clientcmdapi.AuthInfo, desired api.NamedAuthInfo) *clientcmdapi.AuthInfo {
	if desired.Token == "" {
		*cur = clientcmdapi.AuthInfo{
			ClientCertificateData: append([]byte(nil), desired.ClientCertificateData...),
			ClientKeyData:         append([]byte(nil), desired.ClientKeyData...),
		}
	} else {
		*cur = clientcmdapi.AuthInfo{
			Token: desired.Token,
		}
	}
	return cur
}

func setContext(cur *clientcmdapi.Context, desired api.NamedContext) *clientcmdapi.Context {
	*cur = clientcmdapi.Context{
		Cluster:  desired.Cluster,
		AuthInfo: desired.AuthInfo,
	}
	return cur
}

func KubeConfigPath() string {
	return homedir.HomeDir() + "/.kube/config"
}
