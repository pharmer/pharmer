package xorm

import (
	"context"
	"fmt"
	"time"

	"github.com/appscode/go/log"
	"github.com/go-xorm/core"
	"github.com/go-xorm/xorm"
	_ "github.com/lib/pq"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
)

const (
	UID = "xorm"
)

func init() {
	store.RegisterProvider(UID, func(ctx context.Context, cfg *api.PharmerConfig) (store.Interface, error) {
		if cfg.Store.Postgres != nil {
			dbCfg := cfg.Store.Postgres
			log.Debugf("Connecting to %v db on host %v with user %v", dbCfg.DbName, dbCfg.Host, dbCfg.User)
			engine, err := newPGEngine(dbCfg.User, dbCfg.Password, dbCfg.Host, dbCfg.Port, dbCfg.DbName)
			if err != nil {
				return nil, errors.Errorf("failed to connect xorm storage. Reason %v", err)
			}
			return New(engine), nil
		}
		return nil, errors.New("missing store configuration")
	})
}

type XormStore struct {
	engine *xorm.Engine
	owner  string
}

var _ store.Interface = &XormStore{}

func New(engine *xorm.Engine) store.Interface {
	return &XormStore{engine: engine}
}

func (s *XormStore) Owner(id string) store.ResourceInterface {
	ret := *s
	ret.owner = id
	return &ret
}

func (s *XormStore) Credentials() store.CredentialStore {
	return &credentialXormStore{engine: s.engine, owner: s.owner}
}

func (s *XormStore) Clusters() store.ClusterStore {
	return &clusterXormStore{engine: s.engine, owner: s.owner}
}

func (s *XormStore) NodeGroups(cluster string) store.NodeGroupStore {
	return &nodeGroupXormStore{engine: s.engine, cluster: cluster, owner: s.owner}
}

func (s *XormStore) MachineSet(cluster string) store.MachineSetStore {
	return &machineSetXormStore{engine: s.engine, cluster: cluster}
}

func (s *XormStore) Machine(cluster string) store.MachineStore {
	return &machineXormStore{engine: s.engine, cluster: cluster}
}

func (s *XormStore) Certificates(cluster string) store.CertificateStore {
	return &certificateXormStore{engine: s.engine, cluster: cluster, owner: s.owner}
}

func (s *XormStore) SSHKeys(cluster string) store.SSHKeyStore {
	return &sshKeyXormStore{engine: s.engine, cluster: cluster, owner: s.owner}
}

func (s *XormStore) Operations() store.OperationStore {
	fmt.Println("HERER")
	return &operationXormStore{engine: s.engine}
}

// Connects to any databse using provided credentials
func newPGEngine(user, password, host string, port int64, dbName string) (*xorm.Engine, error) {
	cnnstr := fmt.Sprintf("user=%v password=%v host=%v port=%v dbname=%v sslmode=disable",
		user, password, host, port, dbName)
	fmt.Println(cnnstr)
	engine, err := xorm.NewEngine("postgres", cnnstr)
	fmt.Println(err)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	engine.SetMaxIdleConns(0)
	engine.DB().SetConnMaxLifetime(10 * time.Minute)
	// engine.ShowSQL(system.Env() == system.DevEnvironment)
	engine.ShowSQL(true)
	engine.Logger().SetLevel(core.LOG_DEBUG)
	return engine, nil
}
