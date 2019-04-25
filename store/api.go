package store

import (
	"crypto/rsa"
	"crypto/x509"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	apiv1 "github.com/pharmer/pharmer/apis/v1beta1"
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
	List(opts metav1.ListOptions) ([]*apiv1.Credential, error)
	Get(name string) (*apiv1.Credential, error)
	Create(obj *apiv1.Credential) (*apiv1.Credential, error)
	Update(obj *apiv1.Credential) (*apiv1.Credential, error)
	Delete(name string) error
}

type OperationStore interface {
	Get(id string) (*apiv1.Operation, error)
	Update(obj *apiv1.Operation) (*apiv1.Operation, error)
}

type ClusterStore interface {
	List(opts metav1.ListOptions) ([]*apiv1.Cluster, error)
	Get(name string) (*apiv1.Cluster, error)
	Create(obj *apiv1.Cluster) (*apiv1.Cluster, error)
	Update(obj *apiv1.Cluster) (*apiv1.Cluster, error)
	Delete(name string) error
	UpdateStatus(obj *apiv1.Cluster) (*apiv1.Cluster, error)
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
