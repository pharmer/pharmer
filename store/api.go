package store

import (
	"crypto/rsa"
	"crypto/x509"
	"errors"

	"github.com/appscode/pharmer/api"
)

var NotImplemented = errors.New("Not implemented")

type Interface interface {
	Credentials() CredentialStore

	Clusters() ClusterStore
	InstanceGroups(cluster string) InstanceGroupStore
	Instances(cluster string) InstanceStore
	Certificates(cluster string) CertificateStore
	SSHKeys(cluster string) SSHKeyStore
}

type CredentialStore interface {
	List(opts api.ListOptions) ([]*api.Credential, error)
	Get(name string) (*api.Credential, error)
	Create(obj *api.Credential) (*api.Credential, error)
	Update(obj *api.Credential) (*api.Credential, error)
	Delete(name string) error
}

type ClusterStore interface {
	List(opts api.ListOptions) ([]*api.Cluster, error)
	Get(name string) (*api.Cluster, error)
	Create(obj *api.Cluster) (*api.Cluster, error)
	Update(obj *api.Cluster) (*api.Cluster, error)
	Delete(name string) error
	UpdateStatus(obj *api.Cluster) (*api.Cluster, error)
}

type InstanceGroupStore interface {
	List(opts api.ListOptions) ([]*api.InstanceGroup, error)
	Get(name string) (*api.InstanceGroup, error)
	Create(obj *api.InstanceGroup) (*api.InstanceGroup, error)
	Update(obj *api.InstanceGroup) (*api.InstanceGroup, error)
	Delete(name string) error
	UpdateStatus(obj *api.InstanceGroup) (*api.InstanceGroup, error)
}

type InstanceStore interface {
	List(opts api.ListOptions) ([]*api.Instance, error)
	Get(name string) (*api.Instance, error)
	Create(obj *api.Instance) (*api.Instance, error)
	Update(obj *api.Instance) (*api.Instance, error)
	Delete(name string) error
	UpdateStatus(obj *api.Instance) (*api.Instance, error)
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
