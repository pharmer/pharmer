package certificates

import (
	"github.com/appscode/go/crypto/ssh"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	kubeadmconst "k8s.io/kubernetes/cmd/kubeadm/app/constants"
)

func GetPharmerCerts(clusterName string) (*PharmerCertificates, error) {
	pharmerCerts := &PharmerCertificates{}

	cert, key, err := LoadCACertificates(clusterName, kubeadmconst.CACertAndKeyBaseName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load ca Certs")
	}
	pharmerCerts.CACert = CertKeyPair{
		Cert: cert,
		Key:  key,
	}

	cert, key, err = LoadCACertificates(clusterName, kubeadmconst.FrontProxyCACertAndKeyBaseName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load fpca Certs")
	}
	pharmerCerts.FrontProxyCACert = CertKeyPair{
		Cert: cert,
		Key:  key,
	}

	cert, key, err = LoadCACertificates(clusterName, kubeadmconst.ServiceAccountKeyBaseName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load sa keys")
	}
	pharmerCerts.ServiceAccountCert = CertKeyPair{
		Cert: cert,
		Key:  key,
	}

	cert, key, err = LoadCACertificates(clusterName, kubeadmconst.EtcdCACertAndKeyBaseName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load etcd-ca keys")
	}
	pharmerCerts.EtcdCACert = CertKeyPair{
		Cert: cert,
		Key:  key,
	}

	pharmerCerts.SSHKey, err = LoadSSHKey(clusterName, GenSSHKeyName(clusterName))
	if err != nil {
		return nil, errors.Wrap(err, "failed to load ssh keys")
	}

	return pharmerCerts, nil
}

func CreatePharmerCerts(store store.ResourceInterface, cluster *api.Cluster) (*PharmerCertificates, error) {
	pharmerCerts := &PharmerCertificates{}

	cert, key, err := CreateCACertificates(store, cluster.Name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create ca certificates")
	}
	pharmerCerts.CACert = CertKeyPair{
		Cert: cert,
		Key:  key,
	}

	cert, key, err = CreateFrontProxyCACertificates(store, cluster.Name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create fpca certificates")
	}
	pharmerCerts.FrontProxyCACert = CertKeyPair{
		Cert: cert,
		Key:  key,
	}

	cert, key, err = CreateSACertificate(store, cluster.Name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create sa certificates")
	}
	pharmerCerts.ServiceAccountCert = CertKeyPair{
		Cert: cert,
		Key:  key,
	}

	cert, key, err = CreateEtcdCACertificate(store, cluster.Name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create etcd-ca certificates")
	}
	pharmerCerts.EtcdCACert = CertKeyPair{
		Cert: cert,
		Key:  key,
	}

	pubKey, privKey, err := CreateSSHKey(store.SSHKeys(cluster.Name), cluster.GenSSHKeyExternalID())
	if err != nil {
		return nil, errors.Wrap(err, "failed to create ssh keys")
	}
	pharmerCerts.SSHKey = &ssh.SSHKey{
		PublicKey:  pubKey,
		PrivateKey: privKey,
	}

	return pharmerCerts, nil
}

func GenSSHKeyName(clusterName string) string {
	return clusterName + "-sshkey"
}
