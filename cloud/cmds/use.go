package cmds

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/go-term"
	"github.com/appscode/go/io"
	"github.com/appscode/go/log"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/config"
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	clientcmd "k8s.io/client-go/tools/clientcmd/api/v1"
	"k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/homedir"
)

func NewCmdUse() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "use",
		Short:             "Retrieve Kubeconfig for a Kubernetes cluster and change kubectl context",
		Example:           `pharmer cluster use <name>`,
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				log.Fatalln("Missing cluster name")
			}
			if len(args) > 1 {
				log.Fatalln("Multiple cluster name provided.")
			}
			name := args[0]

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
				err = io.CopyFile(bakFile, KubeConfigPath(), 0600)
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

			if konfig.CurrentContext == name+"@pharmer" {
				term.Infoln(fmt.Sprintf("Cluster `%s` is already current context.", name))
				os.Exit(0)
			}

			found := false
			for _, c := range konfig.Contexts {
				if c.Name == name+"@pharmer" {
					found = true
				}
			}
			if !found {
				cfgFile, _ := config.GetConfigFile(cmd.Flags())
				cfg, err := config.LoadConfig(cfgFile)
				if err != nil {
					log.Fatalln(err)
				}
				ctx := cloud.NewContext(context.TODO(), cfg)

				cluster, err := cloud.Store(ctx).Clusters().Get(name)
				if err != nil {
					log.Fatalln(err)
				}
				ctx, err = cloud.LoadCACertificates(ctx, cluster)
				if err != nil {
					log.Fatalln(err)
				}
				adminCert, adminKey, err := cloud.CreateAdminCertificate(ctx)
				if err != nil {
					log.Fatalln(err)
				}
				resp := &proto.ClusterClientConfigResponse{
					ClusterDomain:   cluster.Name + "@pharmer",
					CaCert:          string(cert.EncodeCertPEM(cloud.CACert(ctx))),
					ApiServerUrl:    cluster.APIServerURL(),
					ClusterUserName: cluster.Name + "@pharmer",
					UserCert:        string(cert.EncodeCertPEM(adminCert)),
					UserKey:         string(cert.EncodePrivateKeyPEM(adminKey)),
					ContextName:     cluster.Name + "@pharmer",
				}

				// Upsert cluster
				found := false
				for _, k := range konfig.Clusters {
					if k.Name == resp.ClusterDomain {
						setCluster(&k, resp)
						found = true
						break
					}
				}
				if !found {
					konfig.Clusters = append(konfig.Clusters, *setCluster(&clientcmd.NamedCluster{}, resp))
				}

				// Upsert user
				found = false
				for _, k := range konfig.AuthInfos {
					if k.Name == resp.ClusterUserName {
						setUser(&k, resp)
						found = true
						break
					}
				}
				if !found {
					konfig.AuthInfos = append(konfig.AuthInfos, *setUser(&clientcmd.NamedAuthInfo{}, resp))
				}

				// Upsert context
				found = false
				for _, k := range konfig.Contexts {
					if k.Name == resp.ContextName {
						setContext(&k, resp)
						found = true
						break
					}
				}
				if !found {
					konfig.Contexts = append(konfig.Contexts, *setContext(&clientcmd.NamedContext{}, resp))
				}
			}

			// Update current context
			konfig.CurrentContext = name + "@pharmer"

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
			term.Successln(fmt.Sprintf("kubectl context set to cluster `%s`.", name))
		},
	}
	return cmd
}

func setCluster(c *clientcmd.NamedCluster, resp *proto.ClusterClientConfigResponse) *clientcmd.NamedCluster {
	c.Name = resp.ClusterDomain
	c.Cluster = clientcmd.Cluster{
		CertificateAuthorityData: []byte(resp.CaCert),
		Server: resp.ApiServerUrl,
	}
	return c
}

func setUser(u *clientcmd.NamedAuthInfo, resp *proto.ClusterClientConfigResponse) *clientcmd.NamedAuthInfo {
	u.Name = resp.ClusterUserName
	if resp.UserToken != "" {
		u.AuthInfo = clientcmd.AuthInfo{
			Token: resp.UserToken,
		}
	} else if resp.Password != "" {
		u.AuthInfo = clientcmd.AuthInfo{
			Username: resp.ClusterUserName,
			Password: resp.Password,
		}
	} else {
		u.AuthInfo = clientcmd.AuthInfo{
			ClientCertificateData: []byte(resp.UserCert),
			ClientKeyData:         []byte(resp.UserKey),
		}
	}
	return u
}

func setContext(c *clientcmd.NamedContext, resp *proto.ClusterClientConfigResponse) *clientcmd.NamedContext {
	c.Name = resp.ContextName
	c.Context = clientcmd.Context{
		Cluster:  resp.ClusterDomain,
		AuthInfo: resp.ClusterUserName,
	}
	return c
}

func KubeConfigPath() string {
	return homedir.HomeDir() + "/.kube/config"
}
