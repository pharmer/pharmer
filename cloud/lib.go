package cloud

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/appscode/pharmer/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientcmd "k8s.io/client-go/tools/clientcmd/api/v1"
	"k8s.io/client-go/util/cert"
)

func List(ctx context.Context, opts metav1.ListOptions) ([]*api.Cluster, error) {
	return Store(ctx).Clusters().List(metav1.ListOptions{})
}

func Get(ctx context.Context, name string) (*api.Cluster, error) {
	return Store(ctx).Clusters().Get(name)
}

func Create(ctx context.Context, cluster *api.Cluster) (*api.Cluster, error) {
	if cluster == nil {
		return nil, errors.New("Missing cluster")
	} else if cluster.Name == "" {
		return nil, errors.New("Missing cluster name")
	} else if cluster.Spec.KubernetesVersion == "" {
		return nil, errors.New("Missing cluster version")
	}

	_, err := Store(ctx).Clusters().Get(cluster.Name)
	if err == nil {
		return nil, fmt.Errorf("Cluster exists with name `%s`.", cluster.Name)
	}

	cm, err := GetCloudManager(cluster.Spec.Cloud.CloudProvider, ctx)
	if err != nil {
		return nil, err
	}
	if cluster, err = cm.DefaultSpec(cluster); err != nil {
		return nil, err
	}
	if cluster, err = Store(ctx).Clusters().Create(cluster); err != nil {
		return nil, err
	}

	if ctx, err = CreateCACertificates(ctx, cluster); err != nil {
		return nil, err
	}
	if ctx, err = CreateSSHKey(ctx, cluster); err != nil {
		return nil, err
	}
	if _, err = cm.CreateMasterNodeGroup(cluster); err != nil {
		return nil, err
	}
	if _, err = Store(ctx).Clusters().Update(cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}

func Update(ctx context.Context, cluster *api.Cluster) (*api.Cluster, error) {
	if cluster == nil {
		return nil, errors.New("Missing cluster")
	} else if cluster.Name == "" {
		return nil, errors.New("Missing cluster name")
	} else if cluster.Spec.KubernetesVersion == "" {
		return nil, errors.New("Missing cluster version")
	}

	existing, err := Store(ctx).Clusters().Get(cluster.Name)
	if err != nil {
		return nil, fmt.Errorf("Cluster `%s` does not exist. Reason: %v", cluster.Name, err)
	}
	cluster.Status = existing.Status

	return Store(ctx).Clusters().Update(cluster)
}

func Delete(ctx context.Context, name string) (*api.Cluster, error) {
	if name == "" {
		return nil, errors.New("Missing cluster name")
	}

	cluster, err := Store(ctx).Clusters().Get(name)
	if err != nil {
		return nil, fmt.Errorf("Cluster `%s` does not exist. Reason: %v", name, err)
	}
	cluster.DeletionTimestamp = &metav1.Time{Time: time.Now()}

	return Store(ctx).Clusters().Update(cluster)
}

func GetAdminConfig(ctx context.Context, name string) (*clientcmd.Config, error) {
	cluster, err := Store(ctx).Clusters().Get(name)
	if err != nil {
		return nil, err
	}
	ctx, err = LoadCACertificates(ctx, cluster)
	if err != nil {
		return nil, err
	}
	adminCert, adminKey, err := CreateAdminCertificate(ctx)
	if err != nil {
		return nil, err
	}

	var (
		clusterName = fmt.Sprintf("%s.pharmer", cluster.Name)
		userName    = fmt.Sprintf("cluster-admin@%s.pharmer", cluster.Name)
		ctxName     = fmt.Sprintf("cluster-admin@%s.pharmer", cluster.Name)
	)
	cfg := clientcmd.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Preferences: clientcmd.Preferences{
			Colors: true,
		},
		Clusters: []clientcmd.NamedCluster{
			{
				Name: clusterName,
				Cluster: clientcmd.Cluster{
					Server: cluster.APIServerURL(),
					CertificateAuthorityData: cert.EncodeCertPEM(CACert(ctx)),
				},
			},
		},
		AuthInfos: []clientcmd.NamedAuthInfo{
			{
				Name: userName,
				AuthInfo: clientcmd.AuthInfo{
					ClientCertificateData: cert.EncodeCertPEM(adminCert),
					ClientKeyData:         cert.EncodePrivateKeyPEM(adminKey),
				},
			},
		},
		Contexts: []clientcmd.NamedContext{
			{
				Name: ctxName,
				Context: clientcmd.Context{
					Cluster:  clusterName,
					AuthInfo: userName,
				},
			},
		},
		CurrentContext: ctxName,
	}
	return &cfg, nil
}

func Apply(ctx context.Context, name string, dryRun bool) error {
	if name == "" {
		return errors.New("Missing cluster name")
	}

	cluster, err := Store(ctx).Clusters().Get(name)
	if err != nil {
		return fmt.Errorf("Cluster `%s` does not exist. Reason: %v", name, err)
	}

	cm, err := GetCloudManager(cluster.Spec.Cloud.CloudProvider, ctx)
	if err != nil {
		return err
	}

	return cm.Apply(cluster, dryRun)
}
