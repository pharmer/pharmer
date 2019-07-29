package xorm

import (
	"crypto/rsa"
	"crypto/x509"
	"time"

	"gomodules.xyz/cert"
)

type Certificate struct {
	ID          int64
	Name        string `xorm:"text not null 'name'"`
	ClusterID   int64  `xorm:"bigint not null 'cluster_id'"`
	ClusterName string `xorm:"text not null 'cluster_name'"`
	UID         string `xorm:"text not null 'uid'"`
	Cert        string `xorm:"text NOT NULL 'cert'"`
	Key         string `xorm:"text NOT NULL 'key'"`

	CreationTimestamp time.Time  `xorm:"bigint created 'created_unix'"`
	DateModified      time.Time  `xorm:"bigint updated 'updated_unix'"`
	DeletionTimestamp *time.Time `xorm:"bigint null 'deleted_unix'"`
}

func (Certificate) TableName() string {
	return "ac_cluster_certificate"
}

func encodeCertificate(crt *x509.Certificate, key *rsa.PrivateKey) *Certificate {
	return &Certificate{
		Cert:              string(cert.EncodeCertPEM(crt)),
		Key:               string(cert.EncodePrivateKeyPEM(key)),
		DateModified:      time.Now(),
		DeletionTimestamp: nil,
	}
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
