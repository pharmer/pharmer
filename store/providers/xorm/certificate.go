package xorm

import (
	"crypto/rsa"
	"crypto/x509"
	"time"

	"github.com/go-xorm/xorm"
	"github.com/pkg/errors"
	"gomodules.xyz/cert"
	"gomodules.xyz/secrets/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"pharmer.dev/pharmer/store"
)

type certificateXormStore struct {
	engine  *xorm.Engine
	cluster string
	owner   int64
}

var _ store.CertificateStore = &certificateXormStore{}

func (s *certificateXormStore) Get(name string) (*x509.Certificate, *rsa.PrivateKey, error) {
	if s.cluster == "" {
		return nil, nil, errors.New("missing cluster name")
	}
	if name == "" {
		return nil, nil, errors.New("missing certificate name")
	}

	cluster, err := s.getCluster()
	if err != nil {
		return nil, nil, err
	}

	certificate := &Certificate{
		Name:        name,
		ClusterName: cluster.Name,
		ClusterID:   cluster.ID,
	}
	found, err := s.engine.Get(certificate)
	if err != nil {
		return nil, nil, errors.Errorf("reason: %v", err)
	}
	if !found {
		return nil, nil, errors.Errorf("certificate `%s` does not exist", name)
	}
	crt, err := cert.ParseCertsPEM([]byte(certificate.Cert))
	if err != nil {
		return nil, nil, err
	}
	key, err := cert.ParsePrivateKeyPEM([]byte(certificate.Key.Data))
	if err != nil {
		return nil, nil, err
	}
	return crt[0], key.(*rsa.PrivateKey), nil
}

func (s *certificateXormStore) Create(name string, crt *x509.Certificate, key *rsa.PrivateKey) error {
	if s.cluster == "" {
		return errors.New("missing cluster name")
	}
	if crt == nil {
		return errors.New("missing certificate")
	} else if key == nil {
		return errors.New("missing certificate key")
	}

	cluster, err := s.getCluster()
	if err != nil {
		return err
	}

	certificate := &Certificate{
		Name:        name,
		ClusterName: cluster.Name,
		ClusterID:   cluster.ID,
	}

	found, err := s.engine.Get(certificate)
	if err != nil {
		return err
	}
	if found {
		return errors.Errorf("certificate `%s` already exists", name)
	}

	certificate = &Certificate{
		Name:        name,
		ClusterID:   cluster.ID,
		ClusterName: cluster.Name,
		UID:         string(uuid.NewUUID()),
		Cert:        string(cert.EncodeCertPEM(crt)),
		Key:         types.SecureString{Data: string(cert.EncodePrivateKeyPEM(key))},
		CreatedUnix: time.Now().Unix(),
		DeletedUnix: nil,
	}

	_, err = s.engine.Insert(certificate)

	return err
}

func (s *certificateXormStore) Delete(name string) error {
	if s.cluster == "" {
		return errors.New("missing cluster name")
	}
	if name == "" {
		return errors.New("missing certificate name")
	}

	cluster, err := s.getCluster()
	if err != nil {
		return err
	}

	_, err = s.engine.Delete(&Certificate{Name: name, ClusterName: cluster.Name, ClusterID: cluster.ID})
	return err
}

func (s *certificateXormStore) getCluster() (*Cluster, error) {
	cluster := &Cluster{
		Name:    s.cluster,
		OwnerID: s.owner,
	}
	has, err := s.engine.Get(cluster)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, errors.New("cluster not exists")
	}
	return cluster, nil
}
