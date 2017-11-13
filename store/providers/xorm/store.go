package xorm

import (
	"context"
	"fmt"
	"time"

	"github.com/appscode/go/errors"
	"github.com/appscode/go/log"
	api "github.com/appscode/pharmer/apis/v1alpha1"
	"github.com/appscode/pharmer/store"
	"github.com/go-xorm/core"
	"github.com/go-xorm/xorm"
	_ "github.com/lib/pq"
)

const (
	UID = "xorm"
)

func init() {
	store.RegisterProvider(UID, func(ctx context.Context, cfg *api.PharmerConfig) (store.Interface, error) {
		if cfg.Store.Postgres != nil {
			dbCfg := cfg.Store.Postgres
			log.Debugf("Connecting to %v db on host %v with user %v", dbCfg.Database, dbCfg.Host, dbCfg.User)
			engine, err := newPGEngine(dbCfg.User, dbCfg.Password, dbCfg.Host, dbCfg.Port, dbCfg.Database)
			if err != nil {
				return nil, fmt.Errorf("failed to connect xorm storage. Reason %v", err)
			}
			return New(engine), nil
		}
		return nil, errors.New("missing store configuration")
	})
}

type XormStore struct {
	engine *xorm.Engine
}

var _ store.Interface = &XormStore{}

func New(engine *xorm.Engine) store.Interface {
	return &XormStore{engine: engine}
}

func (s *XormStore) Credentials() store.CredentialStore {
	return &credentialXormStore{engine: s.engine}
}

func (s *XormStore) Clusters() store.ClusterStore {
	return &clusterXormStore{engine: s.engine}
}

func (s *XormStore) NodeGroups(cluster string) store.NodeGroupStore {
	return &nodeGroupXormStore{engine: s.engine, cluster: cluster}
}

func (s *XormStore) Certificates(cluster string) store.CertificateStore {
	return &certificateXormStore{engine: s.engine, cluster: cluster}
}

func (s *XormStore) SSHKeys(cluster string) store.SSHKeyStore {
	return &sshKeyXormStore{engine: s.engine, cluster: cluster}
}

// Connects to any databse using provided credentials
func newPGEngine(user, password, host string, port int64, dbName string) (*xorm.Engine, error) {
	cnnstr := fmt.Sprintf("user=%v password=%v host=%v port=%v dbname=%v sslmode=disable",
		user, password, host, port, dbName)
	engine, err := xorm.NewEngine("postgres", cnnstr)
	if err != nil {
		return nil, errors.FromErr(err).Err()
	}
	engine.SetMaxIdleConns(0)
	engine.DB().SetConnMaxLifetime(10 * time.Minute)
	// engine.ShowSQL(system.Env() == system.DevEnvironment)
	engine.ShowSQL(true)
	engine.Logger().SetLevel(core.LOG_DEBUG)
	return engine, nil
}
