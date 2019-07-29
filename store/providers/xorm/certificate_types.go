package xorm

import (
	"crypto/rsa"
	"crypto/x509"
	"time"

	"gomodules.xyz/cert"
)

type Certificate struct {
	ID          int64 `xorm:"pk autoincr"`
	Name        string
	ClusterID   int64 `xorm:"NOT NULL 'cluster_id'"`
	ClusterName string
	UID         string `xorm:"uid UNIQUE"`
	Cert        string `xorm:"text NOT NULL"`
	Key         string `xorm:"text NOT NULL"`

	CreatedUnix int64  `xorm:"INDEX created"`
	UpdatedUnix int64  `xorm:"INDEX updated"`
	DeletedUnix *int64 `xorm:"null"`
}

func (Certificate) TableName() string {
	return "ac_cluster_certificate"
}

func encodeCertificate(crt *x509.Certificate, key *rsa.PrivateKey) *Certificate {
	return &Certificate{
		Cert:        string(cert.EncodeCertPEM(crt)),
		Key:         string(cert.EncodePrivateKeyPEM(key)),
		UpdatedUnix: time.Now().Unix(),
		DeletedUnix: nil,
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
