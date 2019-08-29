package xorm

import (
	"crypto/rsa"
	"crypto/x509"
	"time"

	"gomodules.xyz/cert"
	"gomodules.xyz/secrets/types"
)

type Certificate struct {
	ID          int64 `xorm:"pk autoincr"`
	Name        string
	ClusterID   int64 `xorm:"NOT NULL 'cluster_id'"`
	ClusterName string
	UID         string `xorm:"uid UNIQUE"`
	Cert        []byte `xorm:"blob NOT NULL"`
	Key         []byte `xorm:"blob NOT NULL"`
	SecretID    string

	CreatedUnix int64  `xorm:"INDEX created"`
	UpdatedUnix int64  `xorm:"INDEX updated"`
	DeletedUnix *int64 `xorm:"null"`
}

func (Certificate) TableName() string {
	return "ac_cluster_certificate"
}

func (certificate *Certificate) FillCertFields(name, uid, clusterName string, clusterId, createdAt int64) {
	certificate.Name = name
	certificate.UID = uid
	certificate.ClusterName = clusterName
	certificate.ClusterID = clusterId
	certificate.CreatedUnix = createdAt
}

func EncodeCertificate(crt *x509.Certificate, key *rsa.PrivateKey) (*Certificate, error) {
	secretId := types.RotateQuarterly()
	certCipher, err := encryptData(secretId, cert.EncodeCertPEM(crt))
	if err != nil {
		log.Error(err, "failed to encrypt certificate")
		return nil, err
	}

	keyCipher, err := encryptData(secretId, cert.EncodePrivateKeyPEM(key))
	if err != nil {
		log.Error(err, "failed to encrypt private key")
		return nil, err
	}

	return &Certificate{
		Cert:        certCipher,
		Key:         keyCipher,
		SecretID:    secretId,
		UpdatedUnix: time.Now().Unix(),
		DeletedUnix: nil,
	}, nil
}

func DecodeCertificate(in *Certificate) (*x509.Certificate, *rsa.PrivateKey, error) {
	certData, err := decryptData(in.SecretID, in.Cert)
	if err != nil {
		log.Error(err, "failed to decrypt certificate")
		return nil, nil, err
	}
	crt, err := cert.ParseCertsPEM(certData)
	if err != nil {
		log.Error(err, "failed to parse cert data")
		return nil, nil, err
	}

	keyData, err := decryptData(in.SecretID, in.Key)
	if err != nil {
		log.Error(err, "failed to decrypt private key")
		return nil, nil, err
	}

	key, err := cert.ParsePrivateKeyPEM(keyData)
	if err != nil {
		log.Error(err, "failed to parse private key data")
		return nil, nil, err
	}
	return crt[0], key.(*rsa.PrivateKey), nil
}
