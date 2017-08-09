package cmds

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	api "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/appctl/pkg/config"
	"github.com/appscode/appctl/pkg/util"
	"github.com/appscode/client/cli"
	term "github.com/appscode/go-term"
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
)

func NewCmdUse() *cobra.Command {
	var req api.ClusterClientConfigRequest

	cmd := &cobra.Command{
		Use:     "use",
		Short:   "Retrieve kubectl configuration for a Kubernetes cluster and change kubectl context",
		Example: `appctl cluster use <name>`,
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				req.Name = args[0]
			} else {
				term.Fatalln("Missing cluster name")
			}
			resp := &api.ClusterClientConfigResponse{}
			var err error
			c := config.ClientOrDie()
			resp, err = c.Kubernetes().V1beta1().Cluster().ClientConfig(c.Context(), &req)
			if err != nil {
				cfg, err := searchLocalKubeConfig(req.Name)
				term.ExitOnError(err)
				if cfg == nil {
					term.Fatalln("Can't find cluster " + req.Name)
				}

				// change current context
				konfig := &KubeConfig{}
				data, _ := ioutil.ReadFile(util.KubeConfigPath())
				yaml.Unmarshal([]byte(data), konfig)
				konfig.CurrentContext = getContextFromClusterName(req.Name)
				output, _ := yaml.Marshal(konfig)
				ioutil.WriteFile(util.KubeConfigPath(), output, 0755)
			} else {
				writeConfig(req.Name, resp)
			}
			term.Infoln("kubectl context set to cluster:", req.Name)
		},
	}
	return cmd
}

func writeConfig(name string, resp *api.ClusterClientConfigResponse) {
	konfig := &KubeConfig{
		APIVersion: "v1",
		Kind:       "Config",
		Preferences: map[string]interface{}{
			"colors": true,
		},
	}
	_, err := os.Stat(util.KubeConfigPath())
	if os.IsNotExist(err) {
		os.MkdirAll(filepath.Dir(util.KubeConfigPath()), 0755)
	}
	if err == nil {
		data, _ := ioutil.ReadFile(util.KubeConfigPath())
		yaml.Unmarshal([]byte(data), konfig)
	}

	// Upsert cluster
	found := false
	for _, k := range konfig.Clusters {
		if k.Name == resp.ClusterDomain {
			setCluster(k, resp)
			found = true
			break
		}
	}
	if !found {
		konfig.Clusters = append(konfig.Clusters, setCluster(&ClustersInfo{}, resp))
	}

	// Upsert user
	found = false
	for _, k := range konfig.Users {
		if k.Name == resp.ClusterUserName {
			setUser(k, resp)
			found = true
			break
		}
	}
	if !found {
		konfig.Users = append(konfig.Users, setUser(&UserInfo{}, resp))
	}

	// Upsert context
	found = false
	for _, k := range konfig.Contexts {
		if k.Name == resp.ContextName {
			setContext(k, resp)
			found = true
			break
		}
	}
	if !found {
		konfig.Contexts = append(konfig.Contexts, setContext(&ContextInfo{}, resp))
	}

	// change current context
	konfig.CurrentContext = resp.ContextName

	output, _ := yaml.Marshal(konfig)
	ioutil.WriteFile(util.KubeConfigPath(), output, 0755)
	term.Infoln("Added cluster configuration to kubectl config")
}

func setCluster(c *ClustersInfo, resp *api.ClusterClientConfigResponse) *ClustersInfo {
	c.Name = resp.ClusterDomain
	c.Cluster = map[string]interface{}{
		"certificate-authority-data": resp.CaCert,
		"server":                     resp.ApiServerUrl,
	}
	return c
}

func setUser(u *UserInfo, resp *api.ClusterClientConfigResponse) *UserInfo {
	u.Name = resp.ClusterUserName
	if resp.UserToken != "" {
		u.User = map[string]interface{}{
			"token": resp.UserToken,
		}
	} else if resp.Password != "" {
		u.User = map[string]interface{}{
			"username": resp.ClusterUserName,
			"password": resp.Password,
		}
	} else {
		u.User = map[string]interface{}{
			"client-certificate-data": resp.UserCert,
			"client-key-data":         resp.UserKey,
		}
	}
	return u
}

func setContext(c *ContextInfo, resp *api.ClusterClientConfigResponse) *ContextInfo {
	c.Name = resp.ContextName
	c.Contextt = map[string]interface{}{
		"cluster": resp.ClusterDomain,
		"user":    resp.ClusterUserName,
	}
	return c
}

type ClustersInfo struct {
	Name    string                 `json:"name"`
	Cluster map[string]interface{} `json:"cluster"`
}

type UserInfo struct {
	Name string                 `json:"name"`
	User map[string]interface{} `json:"user"`
}

type ContextInfo struct {
	Name     string                 `json:"name"`
	Contextt map[string]interface{} `json:"context"`
}

// Adapted from https://github.com/kubernetes/client-go/blob/master/tools/clientcmd/api/v1/types.go#L27
// Simplified to avoid dependency on client-go
type KubeConfig struct {
	Kind           string                 `json:"kind,omitempty"`
	APIVersion     string                 `json:"apiVersion,omitempty"`
	Clusters       []*ClustersInfo        `json:"clusters"`
	Contexts       []*ContextInfo         `json:"contexts"`
	CurrentContext string                 `json:"current-context"`
	Preferences    map[string]interface{} `json:"preferences"`
	Users          []*UserInfo            `json:"users"`
	Extensions     json.RawMessage        `json:"extensions,omitempty"`
}

func getContextFromClusterName(clusterName string) string {
	auth := cli.GetAuthOrDie()
	baseDomain := strings.SplitN(auth.Network.ClusterUrls.BaseAddr, ":", 2)[0]
	if auth.Env.IsHosted() {
		return fmt.Sprintf("%v@%v-%v.%v", auth.UserName, strings.ToLower(clusterName), auth.TeamId, baseDomain)
	}
	return fmt.Sprintf("%v@%v.%v", auth.UserName, strings.ToLower(clusterName), baseDomain)
}
