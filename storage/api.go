package storage

import (
	"errors"

	"github.com/appscode/pharmer/api"
)

var NotImplemented = errors.New("Not implemented")

type Interface interface {
	Credentials() CredentialStore

	Clusters() ClusterStore
	Instances(name string) InstanceStore
	Certificates(name string) CertificateStore
	SSHKeys(name string) SSHKeyStore
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

type InstanceStore interface {
	List(opts api.ListOptions) ([]*api.Instance, error)
	Get(name string) (*api.Instance, error)
	Create(obj *api.Instance) (*api.Instance, error)
	Update(obj *api.Instance) (*api.Instance, error)
	Delete(name string) error
	UpdateStatus(obj *api.Instance) (*api.Instance, error)

	// Deprecated, use Update in a loop
	SaveInstances([]*api.Instance) error
}

type CertificateStore interface {
	Get(name string) (certPEM, keyPEM []byte, err error)
	Create(name string, certPEM, keyPEM []byte) error
	Delete(name string) error
}

type SSHKeyStore interface {
	Get(name string) (pubKey, privKey []byte, err error)
	Create(name string, pubKey, privKey []byte) error
	Delete(name string) error
}
