package cloud

import (
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	cloudapi "github.com/pharmer/cloud/pkg/apis/cloud/v1"
)


type CloudManagerInterface interface {
	GetCluster() *api.Cluster
	GetCaCertPair() *api.CertKeyPair
	GetPharmerCertificates() *api.PharmerCertificates
	GetCredential() (*cloudapi.Credential, error)

	GetAdminClient() (kubernetes.Interface, error)
}

type CloudManager struct {
	Cluster *api.Cluster
	Certs   *api.PharmerCertificates

	Namer       namer
	AdminClient kubernetes.Interface

	Credential *cloudapi.Credential

	owner string
}

func (cm *CloudManager) GetCluster() *api.Cluster {
	return cm.Cluster
}

func (cm *CloudManager) GetCaCertPair() *api.CertKeyPair {
	return &cm.Certs.CACert
}

func (cm *CloudManager) GetPharmerCertificates() *api.PharmerCertificates {
	return cm.Certs
}

func (cm *CloudManager) GetAdminClient() (kubernetes.Interface, error) {
	if cm.AdminClient != nil {
		return cm.AdminClient, nil
	}

	client, err := NewAdminClient(&cm.Certs.CACert, cm.Cluster)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new kube-client")
	}
	cm.AdminClient = client

	return client, nil
}

func (cm *CloudManager) GetCredential() (*cloudapi.Credential, error) {
	if cm.Credential != nil {
		return cm.Credential, nil
	}

	cred, err := store.StoreProvider.Credentials().Get(cm.Cluster.Spec.Config.CredentialName)
	if err != nil {
		return nil, err
	}
	cm.Credential = cred

	return cred, err
}
