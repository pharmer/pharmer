package certificates

import (
	"crypto/rsa"
	"crypto/x509"

	"github.com/appscode/go/crypto/ssh"
)

type Certificates struct {
	CACert             CertKeyPair
	FrontProxyCACert   CertKeyPair
	ServiceAccountCert CertKeyPair
	EtcdCACert         CertKeyPair
	SSHKey             *ssh.SSHKey
}

type CertKeyPair struct {
	Cert *x509.Certificate
	Key  *rsa.PrivateKey
}
