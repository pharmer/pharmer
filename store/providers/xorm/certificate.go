package xorm

import (
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"fmt"
	"time"

	"github.com/appscode/pharmer/phid"
	"github.com/appscode/pharmer/store"
	"github.com/go-xorm/xorm"
)

type CertificateXormStore struct {
	engine  *xorm.Engine
	prefix  string
	cluster string
}

var _ store.CertificateStore = &CertificateXormStore{}

func (s *CertificateXormStore) Get(name string) (*x509.Certificate, *rsa.PrivateKey, error) {
	if s.cluster == "" {
		return nil, nil, errors.New("missing cluster name")
	}
	if name == "" {
		return nil, nil, errors.New("missing certificate name")
	}

	certificate := &Certificate{
		Name:        name,
		ClusterName: s.cluster,
	}
	found, err := s.engine.Get(certificate)
	if !found {
		return nil, nil, fmt.Errorf("certificate `%s` does not exist", name)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("reason: %v", err)
	}

	return decodeCertificate(certificate)
}

func (s *CertificateXormStore) Create(name string, crt *x509.Certificate, key *rsa.PrivateKey) error {
	if s.cluster == "" {
		return errors.New("missing cluster name")
	}
	if crt == nil {
		return errors.New("missing certificate")
	} else if key == nil {
		return errors.New("missing certificate key")
	}

	certificate := &Certificate{
		Name:        name,
		ClusterName: s.cluster,
	}

	found, err := s.engine.Get(certificate)
	if found {
		return fmt.Errorf("certificate `%s` already exists", name)
	}
	if err != nil {
		return err
	}
	certificate, err = encodeCertificate(crt, key)
	if err != nil {
		return err
	}
	certificate.Name = name
	certificate.ClusterName = s.cluster
	certificate.UID = string(phid.NewCert())
	certificate.CreationTimestamp = time.Now()
	_, err = s.engine.Insert(certificate)

	return err
}

func (s *CertificateXormStore) Delete(name string) error {
	if s.cluster == "" {
		return errors.New("missing cluster name")
	}
	if name == "" {
		return errors.New("missing certificate name")
	}

	_, err := s.engine.Delete(&Certificate{Name: name, ClusterName: s.cluster})
	return err
}
