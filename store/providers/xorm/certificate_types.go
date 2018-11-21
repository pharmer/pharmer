package xorm

import (
	"crypto/rsa"
	"crypto/x509"
	"time"

	"k8s.io/client-go/util/cert"
)

type Certificate struct {
	Id                int64
	Name              string     `xorm:"text not null 'name'"`
	ClusterName       string     `xorm:"text not null 'clusterName'"`
	UID               string     `xorm:"text not null 'uid'"`
	Cert              string     `xorm:"text NOT NULL 'cert'"`
	Key               string     `xorm:"text NOT NULL 'key'"`
	CreationTimestamp time.Time  `xorm:"bigint created 'creationTimestamp'"`
	DateModified      time.Time  `xorm:"bigint updated 'dateModified'"`
	DeletionTimestamp *time.Time `xorm:"bigint null 'deletionTimestamp'"`
	ClusterId         int64      `xorm:"bigint not null 'clusterId'"`
}

func (Certificate) TableName() string {
	return `"cluster_certificate"`
}

func encodeCertificate(crt *x509.Certificate, key *rsa.PrivateKey) (*Certificate, error) {
	return &Certificate{
		Cert:              string(cert.EncodeCertPEM(crt)),
		Key:               string(cert.EncodePrivateKeyPEM(key)),
		DateModified:      time.Now(),
		DeletionTimestamp: nil,
	}, nil
}

func decodeCertificate(in *Certificate) (*x509.Certificate, *rsa.PrivateKey, error) {
	crt, err := cert.ParseCertsPEM([]byte(in.Cert))
	if err != nil {
		return nil, nil, err
	}

	key, err := cert.ParsePrivateKeyPEM([]byte(in.Key))
	if err != nil {
		return nil, nil, err
	}
	return crt[0], key.(*rsa.PrivateKey), nil
}
