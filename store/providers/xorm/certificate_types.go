package xorm

import (
	"crypto/rsa"
	"crypto/x509"
	"time"

	"github.com/appscode/pharmer/store"
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
	DeletionTimestamp *time.Time `xorm:"bigint deleted 'deletionTimestamp'"`
}

func (Certificate) TableName() string {
	return `"pharmer"."certificate"`
}

func encodeCertificate(*x509.Certificate, *rsa.PrivateKey, error) (*Certificate, error) {
	return nil, store.ErrNotImplemented
}

func decodeCertificate(in *Certificate) (*x509.Certificate, *rsa.PrivateKey, error) {
	return nil, nil, store.ErrNotImplemented
}
