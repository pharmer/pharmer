package store

import (
	"crypto/rsa"
	"crypto/x509"

	api "github.com/pharmer/pharmer/apis/v1alpha1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var ErrNotImplemented = errors.New("not implemented")

type Interface interface {
	Owner(string) ResourceInterface
	ResourceInterface
}

type ResourceInterface interface {
	Credentials() CredentialStore

	Clusters() ClusterStore
	NodeGroups(cluster string) NodeGroupStore
	Certificates(cluster string) CertificateStore
	SSHKeys(cluster string) SSHKeyStore
}

type CredentialStore interface {
	List(opts metav1.ListOptions) ([]*api.Credential, error)
	Get(name string) (*api.Credential, error)
	Create(obj *api.Credential) (*api.Credential, error)
	Update(obj *api.Credential) (*api.Credential, error)
	Delete(name string) error
}

type ClusterStore interface {
	List(opts metav1.ListOptions) ([]*api.Cluster, error)
	Get(name string) (*api.Cluster, error)
	Create(obj *api.Cluster) (*api.Cluster, error)
	Update(obj *api.Cluster) (*api.Cluster, error)
	Delete(name string) error
	UpdateStatus(obj *api.Cluster) (*api.Cluster, error)
}

type NodeGroupStore interface {
	List(opts metav1.ListOptions) ([]*api.NodeGroup, error)
	Get(name string) (*api.NodeGroup, error)
	Create(obj *api.NodeGroup) (*api.NodeGroup, error)
	Update(obj *api.NodeGroup) (*api.NodeGroup, error)
	Delete(name string) error
	UpdateStatus(obj *api.NodeGroup) (*api.NodeGroup, error)
}

type CertificateStore interface {
	Get(name string) (*x509.Certificate, *rsa.PrivateKey, error)
	Create(name string, crt *x509.Certificate, key *rsa.PrivateKey) error
	Delete(name string) error
}

type SSHKeyStore interface {
	Get(name string) (pubKey, privKey []byte, err error)
	Create(name string, pubKey, privKey []byte) error
	Delete(name string) error
}
