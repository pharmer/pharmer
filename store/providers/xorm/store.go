/*
Copyright The Pharmer Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package xorm

import (
	"fmt"
	"net/url"
	"time"

	api "pharmer.dev/pharmer/apis/v1alpha1"
	"pharmer.dev/pharmer/store"

	"github.com/go-xorm/xorm"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	_ "gocloud.dev/secrets/gcpkms"
	"gomodules.xyz/secrets/types"
	"gomodules.xyz/secrets/xkms"
	"xorm.io/core"
)

const (
	UID = "xorm"
)

func init() {
	store.RegisterProvider(UID, func(cfg *api.PharmerConfig) (store.Interface, error) {
		if cfg.Store.Postgres != nil {
			dbCfg := cfg.Store.Postgres
			engine, err := newPGEngine(dbCfg.User, dbCfg.Password, dbCfg.Host, dbCfg.Port, dbCfg.DbName)
			if err != nil {
				log.Error(err, "failed to connect to xorm storage")
				return nil, err
			}
			log.Info("Connected to database", "db", dbCfg.DbName, "host", dbCfg.Host, "user", dbCfg.User)

			if err := engine.Sync2(tables...); err != nil {
				log.Error(err, "failed to synchronize tables")
				return nil, err
			}

			ds := fmt.Sprintf("user=%v password=%v host=%v port=%v dbname=%v sslmode=disable",
				dbCfg.User, dbCfg.Password, dbCfg.Host, dbCfg.Port, dbCfg.DbName)

			u := url.URL{
				Scheme: xkms.Scheme,
			}
			q := u.Query()
			q.Set("driver", core.POSTGRES)
			q.Set("ds", ds)
			q.Set("master_key_url", dbCfg.MasterKeyURL)
			u.RawQuery = q.Encode()
			if err := xkms.Register(u.String(), engine); err != nil {
				log.Error(err, "failed to register xkms keeper")
				return nil, err
			}
			types.Config(xkms.RotateQuarterly)
			log.Info("xkms keeper registered successfully")
			return New(engine), nil
		}
		return nil, errors.New("missing store configuration")
	})
}

type XormStore struct {
	engine *xorm.Engine
	owner  int64
}

var _ store.Interface = &XormStore{}

func New(engine *xorm.Engine) store.Interface {
	return &XormStore{engine: engine}
}

func (s *XormStore) Owner(id int64) store.ResourceInterface {
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
	return &operationXormStore{engine: s.engine}
}

// Connects to any databse using provided credentials
func newPGEngine(user, password, host string, port int64, dbName string) (*xorm.Engine, error) {
	cnnstr := fmt.Sprintf("user=%v password=%v host=%v port=%v dbname=%v sslmode=disable",
		user, password, host, port, dbName)
	x, err := xorm.NewEngine("postgres", cnnstr)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	x.SetMapper(core.GonicMapper{})
	x.SetMaxIdleConns(0)
	x.DB().SetConnMaxLifetime(10 * time.Minute)
	// engine.ShowSQL(system.Env() == system.DevEnvironment)
	//engine.ShowSQL(true)
	x.Logger().SetLevel(core.LOG_DEBUG)
	return x, nil
}
