package cmds

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/appctl/pkg/config"
	"github.com/appscode/appctl/pkg/util"
	"github.com/appscode/go-term"
	"github.com/appscode/go/flags"
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type ItemList struct {
	Items []map[string]interface{} `json:"items,omitempty"`
}
type backupReq struct {
	sanitize  bool
	backupDir string
	cluster   string
}

func NewCmdBackup() *cobra.Command {
	req := backupReq{}
	cmd := &cobra.Command{
		Use:               "backup",
		Short:             "Takes backup of YAML files of cluster",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			flags.EnsureRequiredFlags(cmd, "cluster", "backup-dir")
			restConfig, err := searchLocalKubeConfig(req.cluster)
			if err != nil || restConfig == nil {
				var clientConfigReq proto.ClusterClientConfigRequest
				clientConfigReq.Name = req.cluster
				c := config.ClientOrDie()
				resp, err := c.Kubernetes().V1beta1().Cluster().ClientConfig(c.Context(), &clientConfigReq)
				util.PrintStatus(err)
				restConfig, err = getConfigFromResp(resp)
				util.PrintStatus(err)
			}
			if err = ensureDirectory(req.backupDir); err != nil {
				term.Fatalln(err)
			}
			req.getAndWriteAllObjectsFromCluster(restConfig)
		},
	}
	cmd.Flags().BoolVar(&req.sanitize, "sanitize", false, " Sanitize fields in YAML")
	cmd.Flags().StringVar(&req.backupDir, "backup-dir", "", "Directory where yaml files will be saved")
	cmd.Flags().StringVar(&req.cluster, "cluster", "", "Name of cluster or Kube config context")
	return cmd
}

func getConfigFromResp(resp *proto.ClusterClientConfigResponse) (*rest.Config, error) {
	var err error
	var tlsCfg rest.TLSClientConfig
	if resp.CaCert != "" {
		tlsCfg.CAData, err = base64.StdEncoding.DecodeString(resp.CaCert)
		if err != nil {
			return nil, err
		}
	}
	if resp.UserCert != "" {
		tlsCfg.CertData, err = base64.StdEncoding.DecodeString(resp.UserCert)
		if err != nil {
			return nil, err
		}
	}
	if resp.UserKey != "" {
		tlsCfg.KeyData, err = base64.StdEncoding.DecodeString(resp.UserKey)
		if err != nil {
			return nil, err
		}
	}

	cfg := &rest.Config{
		Host:            resp.ApiServerUrl,
		TLSClientConfig: tlsCfg,
	}
	if resp.UserToken != "" {
		cfg.BearerToken = resp.UserToken
	} else if len(resp.ClusterUserName) > 0 && len(resp.Password) > 0 {
		cfg.Password = resp.Password
		cfg.Username = resp.ClusterUserName
	}
	return cfg, nil
}

func searchLocalKubeConfig(clusterName string) (*rest.Config, error) {
	apiConfig, err := clientcmd.NewDefaultPathOptions().GetStartingConfig()
	if err != nil {
		return nil, err
	}
	overrides := &clientcmd.ConfigOverrides{CurrentContext: clusterName}
	return clientcmd.NewDefaultClientConfig(*apiConfig, overrides).ClientConfig()
}

func (backup backupReq) getAndWriteAllObjectsFromCluster(kubeConfig *rest.Config) {
	discoveryClient := discovery.NewDiscoveryClientForConfigOrDie(kubeConfig)
	rs, err := discoveryClient.ServerResources()
	if err != nil {
		term.Fatalln(err)
	}

	err = os.MkdirAll(backup.backupDir, 0755)
	if err != nil {
		term.Fatalln(err)
	}
	resBytes, err := yaml.Marshal(rs)
	if err != nil {
		term.Fatalln(err)
	}
	err = ioutil.WriteFile(filepath.Join(backup.backupDir, "api_resources.yaml"), resBytes, 0755)
	if err != nil {
		term.Fatalln(err)
	}

	for _, v := range rs {
		gv, err := schema.ParseGroupVersion(v.GroupVersion)
		if err != nil {
			continue
		}
		for _, rss := range v.APIResources {
			term.Infoln("Taking backup for", rss.Name, "groupversion =", v.GroupVersion)
			if err := rest.SetKubernetesDefaults(kubeConfig); err != nil {
				term.Fatalln(err)
			}
			kubeConfig.ContentConfig = dynamic.ContentConfig()
			kubeConfig.GroupVersion = &schema.GroupVersion{Group: gv.Group, Version: gv.Version}
			kubeConfig.APIPath = "/apis"
			if gv.Group == core.GroupName {
				kubeConfig.APIPath = "/api"
			}
			restClient, err := rest.RESTClientFor(kubeConfig)
			if err != nil {
				term.Fatalln(err)
			}
			request := restClient.Get().Resource(rss.Name).Param("pretty", "true")
			b, err := request.DoRaw()
			if err != nil {
				term.Errorln(err)
				continue
			}
			list := &ItemList{}
			err = yaml.Unmarshal(b, &list)
			if err != nil {
				term.Errorln(err)
				continue
			}
			if len(list.Items) > 1000 {
				ok := term.Ask(fmt.Sprintf("Too many objects (%v). Want to take backup ?", len(list.Items)), true)
				if !ok {
					continue
				}
			}
			for _, ob := range list.Items {
				var selfLink string
				ob["apiVersion"] = v.GroupVersion
				ob["kind"] = rss.Kind
				i, ok := ob["metadata"]
				if ok {
					selfLink = getSelfLinkFromMetadata(i)
				} else {
					term.Errorln("Metadata not found")
					continue
				}
				if backup.sanitize {
					cleanUpObjectMeta(i)
					spec, ok := ob["spec"].(map[string]interface{})
					if ok {
						if rss.Kind == "Pod" {
							spec = cleanUpPodSpec(spec)
						}
						template, ok := spec["template"].(map[string]interface{})
						if ok {
							podSpec, ok := template["spec"].(map[string]interface{})
							if ok {
								template["spec"] = cleanUpPodSpec(podSpec)
							}
						}
					}
					delete(ob, "status")
				}
				b, err := yaml.Marshal(ob)
				if err != nil {
					term.Errorln(err)
					break
				}
				path := filepath.Dir(filepath.Join(backup.backupDir, selfLink))
				obName := filepath.Base(selfLink)
				err = os.MkdirAll(path, 0777)
				if err != nil {
					term.Errorln(err)
					break
				}
				fileName := filepath.Join(path, obName+".yaml")
				if err = ioutil.WriteFile(fileName, b, 0644); err != nil {
					term.Errorln(err)
					continue
				}

			}
		}
	}
}

func ensureDirectory(dir string) error {
	err := os.MkdirAll(dir, 0777)
	return err
}

func cleanUpObjectMeta(i interface{}) {
	meta, ok := i.(map[string]interface{})
	if !ok {
		return
	}
	delete(meta, "creationTimestamp")
	delete(meta, "resourceVersion")
	delete(meta, "selfLink")
	delete(meta, "uid")
	delete(meta, "generateName")
	delete(meta, "generation")
	annotation, ok := meta["annotations"]
	if !ok {
		return
	}
	annotations, ok := annotation.(map[string]string)
	if !ok {
		return
	}
	cleanUpDecorators(annotations)
}

func cleanUpDecorators(i interface{}) {
	m, ok := i.(map[string]interface{})
	if !ok {
		return
	}
	delete(m, "controller-uid")
	delete(m, "deployment.kubernetes.io/desired-replicas")
	delete(m, "deployment.kubernetes.io/max-replicas")
	delete(m, "deployment.kubernetes.io/revision")
	delete(m, "pod-template-hash")
	delete(m, "pv.kubernetes.io/bind-completed")
	delete(m, "pv.kubernetes.io/bound-by-controller")
}

func cleanUpPodSpec(podSpec map[string]interface{}) map[string]interface{} {
	b, err := yaml.Marshal(podSpec)
	if err != nil {
		term.Errorln(err)
		return podSpec
	}
	p := &core.PodSpec{}
	err = yaml.Unmarshal(b, p)
	if err != nil {
		term.Errorln(err)
		return podSpec // Not a podspec
	}
	p.DNSPolicy = core.DNSPolicy("")
	p.NodeName = ""
	if p.ServiceAccountName == "default" {
		p.ServiceAccountName = ""
	}
	p.TerminationGracePeriodSeconds = nil
	for i, c := range p.Containers {
		c.TerminationMessagePath = ""
		p.Containers[i] = c
	}
	for i, c := range p.InitContainers {
		c.TerminationMessagePath = ""
		p.InitContainers[i] = c
	}
	b, err = yaml.Marshal(p)
	if err != nil {
		term.Errorln(err)
		return nil
	}
	var spec map[string]interface{}
	err = yaml.Unmarshal(b, &spec)
	if err != nil {
		term.Errorln(err)
		return spec
	}
	return spec
}

func getSelfLinkFromMetadata(i interface{}) string {
	meta, ok := i.(map[string]interface{})
	if ok {
		return meta["selfLink"].(string)
	}
	return ""
}
