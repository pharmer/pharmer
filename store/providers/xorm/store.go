package xorm

import (
	"context"
	"fmt"
	"time"

	"github.com/appscode/errors"
	"github.com/appscode/log"
	api "github.com/appscode/pharmer/apis/v1alpha1"
	"github.com/appscode/pharmer/store"
	"github.com/go-xorm/core"
	"github.com/go-xorm/xorm"
	_ "github.com/lib/pq"
)

const (
	UID      = "xorm"
	pageSize = 50
	Database = "pharmer"
)

func init() {
	store.RegisterProvider(UID, func(ctx context.Context, cfg *api.PharmerConfig) (store.Interface, error) {
		if cfg.Store.Xorm != nil {
			dbCfg := cfg.Store.Xorm
			log.Debugf("Connecting to %v db on host %v with user %v", Database, dbCfg.Host, dbCfg.User)
			engine, err := newPGEngine(dbCfg.User, dbCfg.Password, dbCfg.Host, dbCfg.Port, Database)
			if err != nil {
				return nil, fmt.Errorf("failed to connect xorm storage. Reason %v", err)
			}
			return &XormStore{engine: engine, prefix: ""}, nil
		}

		return nil, errors.New("missing store configuration")
	})
}

type XormStore struct {
	engine *xorm.Engine
	prefix string
}

var _ store.Interface = &XormStore{}

func (s *XormStore) Credentials() store.CredentialStore {
	return &CredentialXormStore{engine: s.engine, prefix: s.prefix}
}

func (s *XormStore) Clusters() store.ClusterStore {
	return &ClusterXormStore{engine: s.engine, prefix: s.prefix}
}

func (s *XormStore) NodeGroups(cluster string) store.NodeGroupStore {
	return &NodeGroupXormStore{engine: s.engine, prefix: s.prefix, cluster: cluster}
}

func (s *XormStore) Certificates(cluster string) store.CertificateStore {
	return &CertificateXormStore{engine: s.engine, prefix: s.prefix, cluster: cluster}
}

func (s *XormStore) SSHKeys(cluster string) store.SSHKeyStore {
	return &SSHKeyXormStore{engine: s.engine, prefix: s.prefix, cluster: cluster}
}

// Connects to any databse using provided credentials
func newPGEngine(user, password, host string, port int, dbName string) (*xorm.Engine, error) {
	cnnstr := fmt.Sprintf("user=%v password=%v host=%v port=%v dbname=%v sslmode=disable",
		user, password, host, port, dbName)
	engine, err := xorm.NewEngine("postgres", cnnstr)
	if err != nil {
		return nil, errors.FromErr(err).Err()
	}
	// engine.SetMaxOpenConns(2); don't set it causes deadlock.
	engine.SetMaxIdleConns(0)
	engine.DB().SetConnMaxLifetime(10 * time.Minute)
	// engine.ShowSQL(system.Env() == system.DevEnvironment)
	engine.ShowSQL(true)
	engine.Logger().SetLevel(core.LOG_DEBUG)
	return engine, nil
}
