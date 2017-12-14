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
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/cloud/cmds/options"
	"github.com/pharmer/pharmer/config"
	"github.com/spf13/cobra"
	clientcmd "k8s.io/client-go/tools/clientcmd/api/v1"
	"k8s.io/client-go/util/homedir"
)

func NewCmdUse() *cobra.Command {
	clusterConfig := options.NewClusterUseConfig()
	cmd := &cobra.Command{
		Use:               "cluster",
		Short:             "Sets `kubectl` context to given cluster",
		Example:           `pharmer use cluster <name>`,
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if err := clusterConfig.ValidateClusterUseFlags(cmd, args); err != nil {
				term.Fatalln(err)
			}
			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				log.Fatalln(err)
			}
			ctx := cloud.NewContext(context.Background(), cfg, config.GetEnv(cmd.Flags()))

			UseCluster(ctx, clusterConfig)
		},
	}
	clusterConfig.AddClusterUseFlags(cmd.Flags())
	return cmd
}

func UseCluster(ctx context.Context, opt *options.ClusterUseConfig) {
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

	ctxName := fmt.Sprintf("cluster-admin@%s.pharmer", opt.ClusterName)

	if !opt.Overwrite {
		if konfig.CurrentContext == ctxName {
			term.Infoln(fmt.Sprintf("Cluster `%s` is already current context.", opt.ClusterName))
			os.Exit(0)
		}
	}

	found := false
	for _, c := range konfig.Contexts {
		if c.Name == ctxName {
			found = true
		}
	}
	if !found || opt.Overwrite {
		cluster, err := cloud.Store(ctx).Clusters().Get(opt.ClusterName)
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
			if konfig.Clusters[i].Name == c2.Clusters[0].Name {
				setCluster(&konfig.Clusters[i], c2.Clusters[0])
				found = true
				break
			}
		}
		if !found {
			konfig.Clusters = append(konfig.Clusters, *setCluster(&clientcmd.NamedCluster{}, c2.Clusters[0]))
		}

		// Upsert user
		found = false
		for i := range konfig.AuthInfos {
			if konfig.AuthInfos[i].Name == c2.AuthInfos[0].Name {
				setUser(&konfig.AuthInfos[i], c2.AuthInfos[0])
				found = true
				break
			}
		}
		if !found {
			konfig.AuthInfos = append(konfig.AuthInfos, *setUser(&clientcmd.NamedAuthInfo{}, c2.AuthInfos[0]))
		}

		// Upsert context
		found = false
		for i := range konfig.Contexts {
			if konfig.Contexts[i].Name == c2.Contexts[0].Name {
				setContext(&konfig.Contexts[i], c2.Contexts[0])
				found = true
				break
			}
		}
		if !found {
			konfig.Contexts = append(konfig.Contexts, *setContext(&clientcmd.NamedContext{}, c2.Contexts[0]))
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
	term.Successln(fmt.Sprintf("kubectl context set to cluster `%s`.", opt.ClusterName))
}

func setCluster(cur *clientcmd.NamedCluster, desired clientcmd.NamedCluster) *clientcmd.NamedCluster {
	*cur = desired
	return cur
}

func setUser(cur *clientcmd.NamedAuthInfo, desired clientcmd.NamedAuthInfo) *clientcmd.NamedAuthInfo {
	*cur = desired
	return cur
}

func setContext(cur *clientcmd.NamedContext, desired clientcmd.NamedContext) *clientcmd.NamedContext {
	*cur = desired
	return cur
}

func KubeConfigPath() string {
	return homedir.HomeDir() + "/.kube/config"
}
