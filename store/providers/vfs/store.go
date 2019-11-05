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
package vfs

import (
	"fmt"
	"path/filepath"

	api "pharmer.dev/pharmer/apis/v1alpha1"
	"pharmer.dev/pharmer/store"

	"github.com/pkg/errors"
	"gomodules.xyz/stow"
	"gomodules.xyz/stow/local"
)

const (
	UID      = "vfs"
	pageSize = 50
)

func init() {
	store.RegisterProvider(UID, func(cfg *api.PharmerConfig) (store.Interface, error) {
		if cfg.Store.Local != nil {
			stowCfg := stow.ConfigMap{
				local.ConfigKeyPath: filepath.Dir(cfg.Store.Local.Path),
			}
			loc, err := stow.Dial(local.Kind, stowCfg)
			if err != nil {
				return nil, errors.Errorf("failed to connect to local storage. Reason: %v", err)
			}
			name := filepath.Base(cfg.Store.Local.Path)
			container, err := loc.Container(name)
			if err != nil {
				container, err = loc.CreateContainer(name)
				if err != nil {
					return nil, errors.Errorf("failed to open storage container `%s`. Reason: %v", name, err)
				}
			}
			return New(container, ""), nil
		}
		return nil, errors.New("missing store configuration")
	})
}

type FileStore struct {
	container stow.Container
	prefix    string
}

var _ store.Interface = &FileStore{}

func New(container stow.Container, prefix string) store.Interface {
	return &FileStore{container: container, prefix: prefix}
}

func (s *FileStore) Owner(id int64) store.ResourceInterface {
	return s
}

func (s *FileStore) Credentials() store.CredentialStore {
	return &credentialFileStore{container: s.container, prefix: s.prefix}
}

func (s *FileStore) Clusters() store.ClusterStore {
	return &clusterFileStore{container: s.container, prefix: s.prefix}
}

func (s *FileStore) MachineSet(cluster string) store.MachineSetStore {
	return &machineSetFileStore{container: s.container, prefix: s.prefix, cluster: cluster}
}

func (s *FileStore) Machine(cluster string) store.MachineStore {
	return &machineFileStore{container: s.container, prefix: s.prefix, cluster: cluster}
}

func (s *FileStore) Certificates(cluster string) store.CertificateStore {
	return &certificateFileStore{container: s.container, prefix: s.prefix, cluster: cluster}
}

func (s *FileStore) SSHKeys(cluster string) store.SSHKeyStore {
	return &sshKeyFileStore{container: s.container, prefix: s.prefix, cluster: cluster}
}

func (s *FileStore) Operations() store.OperationStore {
	fmt.Println("file operation nil")
	return nil
}
