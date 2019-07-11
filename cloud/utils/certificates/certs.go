package certificates

import (
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	kubeadmconst "k8s.io/kubernetes/cmd/kubeadm/app/constants"
)

func GetPharmerCerts(storeProvider store.ResourceInterface, clusterName string) (*Certificates, error) {
	pharmerCerts := &Certificates{}

	certStore := storeProvider.Certificates(clusterName)
	keyStore := storeProvider.SSHKeys(clusterName)

	cert, key, err := LoadCACertificates(certStore, kubeadmconst.CACertAndKeyBaseName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load ca Certs")
	}
	pharmerCerts.CACert = CertKeyPair{
		Cert: cert,
		Key:  key,
	}

	cert, key, err = LoadCACertificates(certStore, kubeadmconst.FrontProxyCACertAndKeyBaseName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load fpca Certs")
	}
	pharmerCerts.FrontProxyCACert = CertKeyPair{
		Cert: cert,
		Key:  key,
	}

	cert, key, err = LoadCACertificates(certStore, kubeadmconst.ServiceAccountKeyBaseName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load sa keys")
	}
	pharmerCerts.ServiceAccountCert = CertKeyPair{
		Cert: cert,
		Key:  key,
	}

	cert, key, err = LoadCACertificates(certStore, kubeadmconst.EtcdCACertAndKeyBaseName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load etcd-ca keys")
	}
	pharmerCerts.EtcdCACert = CertKeyPair{
		Cert: cert,
		Key:  key,
	}

	pharmerCerts.SSHKey, err = LoadSSHKey(keyStore, GenSSHKeyName(clusterName))
	if err != nil {
		return nil, errors.Wrap(err, "failed to load ssh keys")
	}

	return pharmerCerts, nil
}

func CreateCertsKeys(store store.ResourceInterface, clusterName string) (*Certificates, error) {
	pharmerCerts := &Certificates{}

	certStore := store.Certificates(clusterName)
	keyStore := store.SSHKeys(clusterName)

	cert, key, err := CreateCACertificates(certStore)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create ca certificates")
	}
	pharmerCerts.CACert = CertKeyPair{
		Cert: cert,
		Key:  key,
	}

	cert, key, err = CreateFrontProxyCACertificates(certStore)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create fpca certificates")
	}
	pharmerCerts.FrontProxyCACert = CertKeyPair{
		Cert: cert,
		Key:  key,
	}

	cert, key, err = CreateSACertificate(certStore)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create sa certificates")
	}
	pharmerCerts.ServiceAccountCert = CertKeyPair{
		Cert: cert,
		Key:  key,
	}

	cert, key, err = CreateEtcdCACertificate(certStore)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create etcd-ca certificates")
	}
	pharmerCerts.EtcdCACert = CertKeyPair{
		Cert: cert,
		Key:  key,
	}

	_, _, err = CreateSSHKey(keyStore, GenSSHKeyName(clusterName))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create ssh keys")
	}

	// this properly sets OpensshFingerprint values
	pharmerCerts.SSHKey, err = LoadSSHKey(keyStore, GenSSHKeyName(clusterName))
	if err != nil {
		return nil, errors.Wrap(err, "failed to load ssh keys")
	}

	return pharmerCerts, nil
}

// TODO: move
func GenSSHKeyName(clusterName string) string {
	return clusterName + "-sshkey"
}
