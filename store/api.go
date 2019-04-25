package store

import (
	"crypto/rsa"
	"crypto/x509"

	cloudapi "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

var ErrNotImplemented = errors.New("not implemented")

type Interface interface {
	Owner(string) ResourceInterface
	ResourceInterface
}

type ResourceInterface interface {
	Credentials() CredentialStore

	Operations() OperationStore
	Clusters() ClusterStore
	NodeGroups(cluster string) NodeGroupStore
	Machine(cluster string) MachineStore
	MachineSet(cluster string) MachineSetStore
	Certificates(cluster string) CertificateStore
	SSHKeys(cluster string) SSHKeyStore
}

type CredentialStore interface {
	List(opts metav1.ListOptions) ([]*cloudapi.Credential, error)
	Get(name string) (*cloudapi.Credential, error)
	Create(obj *cloudapi.Credential) (*cloudapi.Credential, error)
	Update(obj *cloudapi.Credential) (*cloudapi.Credential, error)
	Delete(name string) error
}

type OperationStore interface {
	Get(id string) (*api.Operation, error)
	Update(obj *api.Operation) (*api.Operation, error)
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

type MachineStore interface {
	List(opts metav1.ListOptions) ([]*clusterv1.Machine, error)
	Get(name string) (*clusterv1.Machine, error)
	Create(obj *clusterv1.Machine) (*clusterv1.Machine, error)
	Update(obj *clusterv1.Machine) (*clusterv1.Machine, error)
	Delete(name string) error
	UpdateStatus(obj *clusterv1.Machine) (*clusterv1.Machine, error)
}

type MachineSetStore interface {
	List(opts metav1.ListOptions) ([]*clusterv1.MachineSet, error)
	Get(name string) (*clusterv1.MachineSet, error)
	Create(obj *clusterv1.MachineSet) (*clusterv1.MachineSet, error)
	Update(obj *clusterv1.MachineSet) (*clusterv1.MachineSet, error)
	Delete(name string) error
	UpdateStatus(obj *clusterv1.MachineSet) (*clusterv1.MachineSet, error)
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
