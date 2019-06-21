package vfs

import (
	"fmt"
	"path/filepath"

	"github.com/graymeta/stow"
	"github.com/graymeta/stow/local"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
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
