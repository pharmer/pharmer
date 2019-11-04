/*
Copyright The Pharmer Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package store

import (
	"crypto/rsa"
	"crypto/x509"

	cloudapi "pharmer.dev/cloud/pkg/apis/cloud/v1"
	api "pharmer.dev/pharmer/apis/v1alpha1"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

var ErrNotImplemented = errors.New("not implemented")

const (
	vfsUID  = "vfs"
	xormUID = "xorm"
	fakeUID = "fake"
)

type Interface interface {
	Owner(int64) ResourceInterface
	ResourceInterface
}

type ResourceInterface interface {
	Credentials() CredentialStore
	Operations() OperationStore
	Clusters() ClusterStore
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
	UpdateUUID(obj *api.Cluster, uuid string) (*api.Cluster, error)
}

type MachineStore interface {
	List(opts metav1.ListOptions) ([]*clusterapi.Machine, error)
	Get(name string) (*clusterapi.Machine, error)
	Create(obj *clusterapi.Machine) (*clusterapi.Machine, error)
	Update(obj *clusterapi.Machine) (*clusterapi.Machine, error)
	Delete(name string) error
	UpdateStatus(obj *clusterapi.Machine) (*clusterapi.Machine, error)
}

type MachineSetStore interface {
	List(opts metav1.ListOptions) ([]*clusterapi.MachineSet, error)
	Get(name string) (*clusterapi.MachineSet, error)
	Create(obj *clusterapi.MachineSet) (*clusterapi.MachineSet, error)
	Update(obj *clusterapi.MachineSet) (*clusterapi.MachineSet, error)
	Delete(name string) error
	UpdateStatus(obj *clusterapi.MachineSet) (*clusterapi.MachineSet, error)
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
