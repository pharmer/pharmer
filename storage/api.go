package storage

import (
	"crypto/x509"

	"github.com/appscode/pharmer/api"
)

type Store interface {
	Clusters() ClusterStore
	Instances() InstanceStore
	Credentials() CredentialStore
	Certificates() CertificateStore
}

type ClusterStore interface {
	GetActiveCluster(name string) ([]*api.Cluster, error)
	LoadCluster(name string) (*api.Cluster, error)
	SaveCluster(*api.Cluster) error
}

type InstanceStore interface {
	LoadInstance(name string) (*api.Instance, error)
	LoadInstances(cluster string) ([]*api.Instance, error)
	SaveInstance(instance *api.Instance) error
	SaveInstances([]*api.Instance) error
}

type CertificateStore interface {
	LoadCertificate(name string) (*x509.Certificate, error)
	SaveCertificate(cert *x509.Certificate) error
}

type CredentialStore interface {
	// Load(name string) (*api.Cluster, error)
	// Save(*api.Cluster) error
}
