package cloud

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/x509"

	"github.com/appscode/go/crypto/ssh"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	"gomodules.xyz/cert"
	kubeadmconst "k8s.io/kubernetes/cmd/kubeadm/app/constants"
)

func CreateCACertificates(storeProvider store.ResourceInterface, clusterName string) (*x509.Certificate, *rsa.PrivateKey, error) {
	return CreateCertificates(storeProvider, clusterName, api.CACertName, api.CACertCommonName)
}

func CreateFrontProxyCACertificates(storeProvider store.ResourceInterface, clusterName string) (*x509.Certificate, *rsa.PrivateKey, error) {
	return CreateCertificates(storeProvider, clusterName, api.FrontProxyCACertName, api.FrontProxyCACertCommonName)
}

func CreateSACertificate(storeProvider store.ResourceInterface, clusterName string) (*x509.Certificate, *rsa.PrivateKey, error) {
	return CreateCertificates(storeProvider, clusterName, api.SAKeyName, api.SAKeyCommonName)
}

func CreateEtcdCACertificate(storeProvider store.ResourceInterface, clusterName string) (*x509.Certificate, *rsa.PrivateKey, error) {
	return CreateCertificates(storeProvider, clusterName, api.ETCDCACertName, api.ETCDCACertCommonName)
}

func CreateCertificates(storeProvider store.ResourceInterface, clusterName, name, commonName string) (*x509.Certificate, *rsa.PrivateKey, error) {
	certStore := storeProvider.Certificates(clusterName)

	caKey, err := cert.NewPrivateKey()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to generate private key")
	}
	caCert, err := cert.NewSelfSignedCACert(cert.Config{CommonName: commonName}, caKey)
	if err != nil {
		return nil, nil, errors.Errorf("failed to generate self-signed certificate. Reason: %v", err)
	}

	if err = certStore.Create(name, caCert, caKey); err != nil {
		return nil, nil, errors.Wrapf(err, "failed to create %q certificate", name)
	}

	return caCert, caKey, nil
}

func LoadCACertificates(clusterName, name string) (*x509.Certificate, *rsa.PrivateKey, error) {
	certStore := store.StoreProvider.Certificates(clusterName)

	caCert, caKey, err := certStore.Get(name)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get CA certificates")
	}
	return caCert, caKey, nil
}

func CreateAdminCertificate(caCert *x509.Certificate, caKey crypto.Signer) (*x509.Certificate, *rsa.PrivateKey, error) {
	cfg := cert.Config{
		CommonName:   "cluster-admin",
		Organization: []string{kubeadmconst.SystemPrivilegedGroup},
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	adminKey, err := cert.NewPrivateKey()
	if err != nil {
		return nil, nil, errors.Errorf("failed to generate private key. Reason: %v", err)
	}

	adminCert, err := cert.NewSignedCert(cfg, adminKey, caCert, caKey)
	if err != nil {
		return nil, nil, errors.Errorf("failed to generate server certificate. Reason: %v", err)
	}
	return adminCert, adminKey, nil
}

func GetAdminCertificate(ctx context.Context, cluster *api.Cluster, owner string) (*x509.Certificate, *rsa.PrivateKey, error) {
	certStore := store.StoreProvider.Certificates(cluster.Name)
	admCert, admKey, err := certStore.Get("admin")
	if err != nil {
		return nil, nil, errors.Errorf("failed to get admin certificates. Reason: %v", err)
	}
	return admCert, admKey, nil
}

func CreateSSHKey(storeProvider store.SSHKeyStore, name string) ([]byte, []byte, error) {
	sshKey, err := ssh.NewSSHKeyPair()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create new ssh key pair")
	}

	err = storeProvider.Create(name, sshKey.PublicKey, sshKey.PrivateKey)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to store ssh keys")
	}

	return sshKey.PublicKey, sshKey.PrivateKey, nil
}

func LoadSSHKey(clusterName, keyName string) (*ssh.SSHKey, error) {
	publicKey, privateKey, err := store.StoreProvider.SSHKeys(clusterName).Get(keyName)
	if err != nil {
		return nil, errors.Errorf("failed to get SSH key. Reason: %v", err)
	}

	sshkeys, err := ssh.ParseSSHKeyPair(string(publicKey), string(privateKey))
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse SSH key")
	}

	return sshkeys, err
}
