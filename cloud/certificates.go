package cloud

import (
	"crypto/rsa"
	"crypto/x509"
)

type PharmerCertificates struct {
	CACert             CertKeyPair
	FrontProxyCACert   CertKeyPair
	ServiceAccountCert CertKeyPair
	EtcdCACert         CertKeyPair
	SSHKey             SSHKey
}

type CertKeyPair struct {
	Cert *x509.Certificate
	Key  *rsa.PrivateKey
}

type SSHKey struct {
	PublicKey  []byte
	PrivateKey []byte
}
