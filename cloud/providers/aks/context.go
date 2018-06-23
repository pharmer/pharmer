package aks

import (
	"context"
	"sync"

	api "github.com/pharmer/pharmer/apis/v1alpha1"
	. "github.com/pharmer/pharmer/cloud"
	"k8s.io/client-go/kubernetes"
	//"fmt"
	"k8s.io/client-go/rest"
	//"k8s.io/client-go/util/cert"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/ghodss/yaml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientcmd "k8s.io/client-go/tools/clientcmd/api/v1"
)

type ClusterManager struct {
	ctx     context.Context
	cluster *api.Cluster
	conn    *cloudConnector
	// Deprecated
	namer namer
	m     sync.Mutex
}

var _ Interface = &ClusterManager{}

const (
	UID              = "aks"
	RoleClusterUser  = "clusterUser"
	RoleClusterAdmin = "clusterAdmin"
)

func init() {
	RegisterCloudManager(UID, func(ctx context.Context) (Interface, error) { return New(ctx), nil })
}

func New(ctx context.Context) Interface {
	return &ClusterManager{ctx: ctx}
}

type paramK8sClient struct{}

func (cm *ClusterManager) GetAdminClient() (kubernetes.Interface, error) {
	cm.m.Lock()
	defer cm.m.Unlock()

	v := cm.ctx.Value(paramK8sClient{})
	if kc, ok := v.(kubernetes.Interface); ok && kc != nil {
		return kc, nil
	}

	kc, err := cm.GetAKSAdminClient()
	if err != nil {
		return nil, err
	}
	cm.ctx = context.WithValue(cm.ctx, paramK8sClient{}, kc)
	return kc, nil
}

func (cm *ClusterManager) GetAKSAdminClient() (kubernetes.Interface, error) {
	resp, err := cm.conn.managedClient.GetAccessProfiles(context.Background(), cm.namer.ResourceGroupName(), cm.cluster.Name, RoleClusterUser)
	if err != nil {
		return nil, err
	}
	kubeconfig, err := base64.StdEncoding.DecodeString(*resp.KubeConfig)
	if err != nil {
		return nil, err
	}
	kubeconfig, err = yaml.YAMLToJSON(kubeconfig)
	if err != nil {
		return nil, err
	}
	var konfig clientcmd.Config
	err = json.Unmarshal(kubeconfig, &konfig)
	if err != nil {
		return nil, err
	}

	cfg := &rest.Config{
		Host: konfig.Clusters[0].Cluster.Server,
		TLSClientConfig: rest.TLSClientConfig{
			CAData:   konfig.Clusters[0].Cluster.CertificateAuthorityData,
			CertData: konfig.AuthInfos[0].AuthInfo.ClientCertificateData,
			KeyData:  konfig.AuthInfos[0].AuthInfo.ClientKeyData,
		},
	}
	return kubernetes.NewForConfig(cfg)

}

func (cm *ClusterManager) GetKubeConfig(cluster *api.Cluster) (*api.KubeConfig, error) {
	var err error
	cm.cluster = cluster
	cm.namer = namer{cluster: cm.cluster}
	if cm.conn, err = NewConnector(cm.ctx, cm.cluster); err != nil {
		return nil, err
	}

	resp, err := cm.conn.managedClient.GetAccessProfiles(context.Background(), cm.namer.ResourceGroupName(), cm.cluster.Name, RoleClusterUser)
	if err != nil {
		return nil, err
	}
	kubeconfig, err := base64.StdEncoding.DecodeString(*resp.KubeConfig)
	if err != nil {
		return nil, err
	}
	kubeconfig, err = yaml.YAMLToJSON(kubeconfig)
	if err != nil {
		return nil, err
	}

	var konfig clientcmd.Config
	err = json.Unmarshal(kubeconfig, &konfig)
	if err != nil {
		return nil, err
	}
	var (
		clusterName = fmt.Sprintf("%s.pharmer", cluster.Name)
		userName    = fmt.Sprintf("cluster-admin@%s.pharmer", cluster.Name)
		ctxName     = fmt.Sprintf("cluster-admin@%s.pharmer", cluster.Name)
	)
	cfg := api.KubeConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "KubeConfig",
		},
		Preferences: api.Preferences{
			Colors: true,
		},
		Cluster: api.NamedCluster{
			Name:   clusterName,
			Server: konfig.Clusters[0].Cluster.Server,
			CertificateAuthorityData: konfig.Clusters[0].Cluster.CertificateAuthorityData,
		},
		AuthInfo: api.NamedAuthInfo{
			Name: userName,
			ClientCertificateData: konfig.AuthInfos[0].AuthInfo.ClientCertificateData,
			ClientKeyData:         konfig.AuthInfos[0].AuthInfo.ClientKeyData,
		},
		Context: api.NamedContext{
			Name:     ctxName,
			Cluster:  clusterName,
			AuthInfo: userName,
		},
	}
	return &cfg, nil
}
