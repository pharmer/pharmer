package cmds

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	ioutilz "github.com/appscode/go/ioutil"
	"github.com/appscode/go/log"
	"github.com/appscode/go/term"
	"github.com/ghodss/yaml"
	api "github.com/pharmer/pharmer/apis/v1alpha1"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/cloud/cmds/options"
	"github.com/pharmer/pharmer/config"
	"github.com/spf13/cobra"
	clientcmd "k8s.io/client-go/tools/clientcmd/api/v1"
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

			UseCluster(ctx, opts)
		},
	}
	opts.AddFlags(cmd.Flags())
	return cmd
}

func UseCluster(ctx context.Context, opts *options.ClusterUseConfig) {
	var konfig clientcmd.Config
	if _, err := os.Stat(KubeConfigPath()); err == nil {
		// ~/.kube/config exists
		data, err := ioutil.ReadFile(KubeConfigPath())
		if err != nil {
			log.Fatalln(err)
		}
		data, err = yaml.YAMLToJSON(data)
		if err != nil {
			log.Fatalln(err)
		}
		err = json.Unmarshal(data, &konfig)
		if err != nil {
			log.Fatalln(err)
		}

		bakFile := KubeConfigPath() + ".bak." + time.Now().Format("2006-01-02T15-04")
		err = ioutilz.CopyFile(bakFile, KubeConfigPath(), 0600)
		if err != nil {
			log.Fatalln(err)
		}
		term.Infoln(fmt.Sprintf("Current Kubeconfig is backed up as %s.", bakFile))
	} else {
		konfig = clientcmd.Config{
			APIVersion: "v1",
			Kind:       "Config",
			Preferences: clientcmd.Preferences{
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

	found := false
	for _, c := range konfig.Contexts {
		if c.Name == ctxName {
			found = true
		}
	}
	if !found || opts.Overwrite {
		cluster, err := cloud.Store(ctx).Clusters().Get(opts.ClusterName)
		if err != nil {
			term.Fatalln(err)
		}
		c2, err := cloud.GetAdminConfig(ctx, cluster)
		if err != nil {
			log.Fatalln(err)
		}

		// Upsert cluster
		found := false
		for i := range konfig.Clusters {
			if konfig.Clusters[i].Name == c2.Cluster.Name {
				setCluster(&konfig.Clusters[i], c2.Cluster)
				found = true
				break
			}
		}
		if !found {
			konfig.Clusters = append(konfig.Clusters, *setCluster(&clientcmd.NamedCluster{}, c2.Cluster))
		}

		// Upsert user
		found = false
		for i := range konfig.AuthInfos {
			if konfig.AuthInfos[i].Name == c2.AuthInfo.Name {
				setUser(&konfig.AuthInfos[i], c2.AuthInfo)
				found = true
				break
			}
		}
		if !found {
			konfig.AuthInfos = append(konfig.AuthInfos, *setUser(&clientcmd.NamedAuthInfo{}, c2.AuthInfo))
		}

		// Upsert context
		found = false
		for i := range konfig.Contexts {
			if konfig.Contexts[i].Name == c2.Context.Name {
				setContext(&konfig.Contexts[i], c2.Context)
				found = true
				break
			}
		}
		if !found {
			konfig.Contexts = append(konfig.Contexts, *setContext(&clientcmd.NamedContext{}, c2.Context))
		}
	}

	// Update current context
	konfig.CurrentContext = ctxName

	err := os.MkdirAll(filepath.Dir(KubeConfigPath()), 0755)
	if err != nil {
		log.Fatalln(err)
	}
	data, err := yaml.Marshal(konfig)
	if err != nil {
		log.Fatalln(err)
	}
	err = ioutil.WriteFile(KubeConfigPath(), data, 0600)
	if err != nil {
		log.Fatalln(err)
	}
	term.Successln(fmt.Sprintf("kubectl context set to cluster `%s`.", opts.ClusterName))
}

func setCluster(cur *clientcmd.NamedCluster, desired api.NamedCluster) *clientcmd.NamedCluster {
	d := clientcmd.NamedCluster{
		Name: desired.Name,
		Cluster: clientcmd.Cluster{
			Server: desired.Server,
			CertificateAuthorityData: append([]byte(nil), desired.CertificateAuthorityData...),
		},
	}
	*cur = d
	return cur
}

func setUser(cur *clientcmd.NamedAuthInfo, desired api.NamedAuthInfo) *clientcmd.NamedAuthInfo {
	d := clientcmd.NamedAuthInfo{
		Name: desired.Name,
	}
	if desired.Token == "" {
		d.AuthInfo = clientcmd.AuthInfo{
			ClientCertificateData: append([]byte(nil), desired.ClientCertificateData...),
			ClientKeyData:         append([]byte(nil), desired.ClientKeyData...),
		}
	} else {
		d.AuthInfo = clientcmd.AuthInfo{
			Token: desired.Token,
		}
	}
	*cur = d
	return cur
}

func setContext(cur *clientcmd.NamedContext, desired api.NamedContext) *clientcmd.NamedContext {
	d := clientcmd.NamedContext{
		Name: desired.Name,
		Context: clientcmd.Context{
			Cluster:  desired.Cluster,
			AuthInfo: desired.AuthInfo,
		},
	}
	*cur = d
	return cur
}

func KubeConfigPath() string {
	return homedir.HomeDir() + "/.kube/config"
}
