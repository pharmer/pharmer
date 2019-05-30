package cloud

import (
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	api "github.com/pharmer/pharmer/apis/v1beta1"
)


type CloudManagerInterface interface {
	GetCluster() *api.Cluster
	GetCaCertPair() *api.CertKeyPair
	GetPharmerCertificates() *api.PharmerCertificates

	GetAdminClient() (kubernetes.Interface, error)
}

type CloudManager struct {
	Cluster *api.Cluster
	Certs   *api.PharmerCertificates

	namer       namer
	adminClient kubernetes.Interface

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
	if cm.adminClient != nil {
		return cm.adminClient, nil
	}

	client, err := NewAdminClient(&cm.Certs.CACert, cm.Cluster)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new kube-client")
	}
	cm.adminClient = client

	return client, nil
}
