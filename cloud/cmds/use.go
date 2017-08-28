package cmds

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/pharmer/api"
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
)

func NewCmdUse() *cobra.Command {
	var req proto.ClusterClientConfigRequest

	cmd := &cobra.Command{
		Use:               "use",
		Short:             "Retrieve kubectl configuration for a Kubernetes cluster and change kubectl context",
		Example:           `appctl cluster use <name>`,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				req.Name = args[0]
			} else {
				return errors.New("Missing cluster name")
			}
			resp := &proto.ClusterClientConfigResponse{}
			var err error
			resp, err = clientConfig(&req)
			if err != nil {
				cfg, err := searchLocalKubeConfig(req.Name)
				if err != nil {
					return err
				}
				if cfg == nil {
					return errors.New("Can't find cluster " + req.Name)
				}

				// change current context
				konfig := &api.KubeConfig{}
				data, _ := ioutil.ReadFile(KubeConfigPath())
				yaml.Unmarshal([]byte(data), konfig)
				// konfig.CurrentContext = getContextFromClusterName(req.Name) // TODO: FixIt!
				output, _ := yaml.Marshal(konfig)
				ioutil.WriteFile(KubeConfigPath(), output, 0755)
			} else {
				writeConfig(req.Name, resp)
			}
			fmt.Println("kubectl context set to cluster:", req.Name)
			return nil
		},
	}
	return cmd
}

func clientConfig(in *proto.ClusterClientConfigRequest) (*proto.ClusterClientConfigResponse, error) {
	return nil, nil
}

func writeConfig(name string, resp *proto.ClusterClientConfigResponse) {
	konfig := &api.KubeConfig{
		APIVersion: "v1",
		Kind:       "Config",
		Preferences: map[string]interface{}{
			"colors": true,
		},
	}
	_, err := os.Stat(KubeConfigPath())
	if os.IsNotExist(err) {
		os.MkdirAll(filepath.Dir(KubeConfigPath()), 0755)
	}
	if err == nil {
		data, _ := ioutil.ReadFile(KubeConfigPath())
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
		konfig.Clusters = append(konfig.Clusters, setCluster(&api.ClustersInfo{}, resp))
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
		konfig.Users = append(konfig.Users, setUser(&api.UserInfo{}, resp))
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
		konfig.Contexts = append(konfig.Contexts, setContext(&api.ContextInfo{}, resp))
	}

	// change current context
	konfig.CurrentContext = resp.ContextName

	output, _ := yaml.Marshal(konfig)
	ioutil.WriteFile(KubeConfigPath(), output, 0755)
	fmt.Println("Added cluster configuration to kubectl config")
}

func setCluster(c *api.ClustersInfo, resp *proto.ClusterClientConfigResponse) *api.ClustersInfo {
	c.Name = resp.ClusterDomain
	c.Cluster = map[string]interface{}{
		"certificate-authority-data": resp.CaCert,
		"server":                     resp.ApiServerUrl,
	}
	return c
}

func setUser(u *api.UserInfo, resp *proto.ClusterClientConfigResponse) *api.UserInfo {
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

func setContext(c *api.ContextInfo, resp *proto.ClusterClientConfigResponse) *api.ContextInfo {
	c.Name = resp.ContextName
	c.Contextt = map[string]interface{}{
		"cluster": resp.ClusterDomain,
		"user":    resp.ClusterUserName,
	}
	return c
}
