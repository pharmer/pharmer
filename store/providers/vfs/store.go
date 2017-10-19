package vfs

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"

	api "github.com/appscode/pharmer/apis/v1alpha1"
	"github.com/appscode/pharmer/credential"
	"github.com/appscode/pharmer/store"
	"github.com/graymeta/stow"
	"github.com/graymeta/stow/azure"
	"github.com/graymeta/stow/google"
	"github.com/graymeta/stow/local"
	"github.com/graymeta/stow/s3"
	"github.com/graymeta/stow/swift"
)

const (
	UID      = "vfs"
	pageSize = 50
)

func init() {
	store.RegisterProvider(UID, func(ctx context.Context, cfg *api.PharmerConfig) (store.Interface, error) {
		if cfg.Store.Local != nil {
			stowCfg := stow.ConfigMap{
				local.ConfigKeyPath: filepath.Dir(cfg.Store.Local.Path),
			}
			loc, err := stow.Dial(local.Kind, stowCfg)
			if err != nil {
				return nil, fmt.Errorf("Failed to connect to local storage. Reason: %v.", err)
			}
			name := filepath.Base(cfg.Store.Local.Path)
			container, err := loc.Container(name)
			if err != nil {
				container, err = loc.CreateContainer(name)
				if err != nil {
					return nil, fmt.Errorf("Failed to open storage container `%s`. Reason: %v.", name, err)
				}
			}
			return &FileStore{container: container, prefix: ""}, nil
		} else if cfg.Store.S3 != nil {
			cred, err := cfg.GetCredential(cfg.Store.CredentialName)
			if err != nil {
				return nil, err
			}
			stowCfg := stow.ConfigMap{
				s3.ConfigAccessKeyID: cred.Spec.Data[credential.AWSAccessKeyID],
				s3.ConfigEndpoint:    cfg.Store.S3.Endpoint,
				s3.ConfigRegion:      "us-east-1", // only used for creating buckets
				s3.ConfigSecretKey:   cred.Spec.Data[credential.AWSSecretAccessKey],
			}
			if u, err := url.Parse(cfg.Store.S3.Endpoint); err == nil {
				stowCfg[s3.ConfigDisableSSL] = strconv.FormatBool(u.Scheme == "http")
			}
			loc, err := stow.Dial(s3.Kind, stowCfg)
			if err != nil {
				return nil, fmt.Errorf("Failed to connect to S3 storage. Reason: %v.", err)
			}
			name := cfg.Store.S3.Bucket
			container, err := loc.Container(name)
			if err != nil {
				container, err = loc.CreateContainer(name)
				if err != nil {
					return nil, fmt.Errorf("Failed to open storage container `%s`. Reason: %v.", name, err)
				}
			}
			return &FileStore{container: container, prefix: cfg.Store.S3.Prefix}, nil
		} else if cfg.Store.GCS != nil {
			cred, err := cfg.GetCredential(cfg.Store.CredentialName)
			if err != nil {
				return nil, err
			}
			stowCfg := stow.ConfigMap{
				google.ConfigProjectId: cred.Spec.Data[credential.GCEProjectID],
				google.ConfigJSON:      cred.Spec.Data[credential.GCEServiceAccount],
			}
			loc, err := stow.Dial(google.Kind, stowCfg)
			if err != nil {
				return nil, fmt.Errorf("Failed to connect to GCS storage. Reason: %v.", err)
			}
			container, err := loc.Container(cfg.Store.GCS.Bucket)
			if err != nil {
				return nil, fmt.Errorf("Failed to open storage container `%s`. Reason: %v.", cfg.Store.GCS.Bucket, err)
			}
			return &FileStore{container: container, prefix: cfg.Store.GCS.Prefix}, nil
		} else if cfg.Store.Azure != nil {
			cred, err := cfg.GetCredential(cfg.Store.CredentialName)
			if err != nil {
				return nil, err
			}
			stowCfg := stow.ConfigMap{
				azure.ConfigAccount: cred.Spec.Data[credential.AzureStorageAccount],
				azure.ConfigKey:     cred.Spec.Data[credential.AzureStorageKey],
			}
			loc, err := stow.Dial(azure.Kind, stowCfg)
			if err != nil {
				return nil, fmt.Errorf("Failed to connect to Azure storage. Reason: %v.", err)
			}
			name := cfg.Store.Azure.Container
			container, err := loc.Container(name)
			if err != nil {
				container, err = loc.CreateContainer(name)
				if err != nil {
					return nil, fmt.Errorf("Failed to open storage container `%s`. Reason: %v.", name, err)
				}
			}
			return &FileStore{container: container, prefix: cfg.Store.Azure.Prefix}, nil
		} else if cfg.Store.Swift != nil {
			cred, err := cfg.GetCredential(cfg.Store.CredentialName)
			if err != nil {
				return nil, err
			}
			stowCfg := stow.ConfigMap{}

			// https://github.com/restic/restic/blob/master/src/restic/backend/swift/config.go
			for _, val := range []struct {
				stowKey string
				jsonKey string
			}{
				// v2/v3 specific
				{swift.ConfigUsername, credential.SwiftUsername},
				{swift.ConfigKey, credential.SwiftKey},
				{swift.ConfigRegion, credential.SwiftRegion},
				{swift.ConfigTenantAuthURL, credential.SwiftTenantAuthURL},

				// v3 specific
				{swift.ConfigDomain, credential.SwiftDomain},
				{swift.ConfigTenantName, credential.SwiftTenantName},
				{swift.ConfigTenantDomain, credential.SwiftTenantDomain},

				// v2 specific
				{swift.ConfigTenantId, credential.SwiftTenantId},
				{swift.ConfigTenantName, credential.SwiftTenantName},

				// v1 specific
				{swift.ConfigTenantAuthURL, credential.SwiftTenantAuthURL},
				{swift.ConfigUsername, credential.SwiftUsername},
				{swift.ConfigKey, credential.SwiftKey},

				// Manual authentication
				{swift.ConfigStorageURL, credential.SwiftStorageURL},
				{swift.ConfigAuthToken, credential.SwiftAuthToken},
			} {
				if _, exists := stowCfg[val.stowKey]; !exists {
					stowCfg[val.stowKey] = cred.Spec.Data[val.jsonKey]
				}
			}

			loc, err := stow.Dial(swift.Kind, stowCfg)
			if err != nil {
				return nil, fmt.Errorf("Failed to connect to Swift storage. Reason: %v.", err)
			}
			name := cfg.Store.Swift.Container
			container, err := loc.Container(name)
			if err != nil {
				container, err = loc.CreateContainer(name)
				if err != nil {
					return nil, fmt.Errorf("Failed to open storage container `%s`. Reason: %v.", name, err)
				}
			}
			return &FileStore{container: container, prefix: cfg.Store.Swift.Prefix}, nil
		}
		return nil, errors.New("Missing store configuration")
	})
}

type FileStore struct {
	container stow.Container
	prefix    string
}

var _ store.Interface = &FileStore{}

func (s *FileStore) Credentials() store.CredentialStore {
	return &CredentialFileStore{container: s.container, prefix: s.prefix}
}

func (s *FileStore) Clusters() store.ClusterStore {
	return &ClusterFileStore{container: s.container, prefix: s.prefix}
}

func (s *FileStore) NodeGroups(cluster string) store.NodeGroupStore {
	return &NodeGroupFileStore{container: s.container, prefix: s.prefix, cluster: cluster}
}

func (s *FileStore) Instances(cluster string) store.InstanceStore {
	return &InstanceFileStore{container: s.container, prefix: s.prefix, cluster: cluster}
}

func (s *FileStore) Certificates(cluster string) store.CertificateStore {
	return &CertificateFileStore{container: s.container, prefix: s.prefix, cluster: cluster}
}

func (s *FileStore) SSHKeys(cluster string) store.SSHKeyStore {
	return &SSHKeyFileStore{container: s.container, prefix: s.prefix, cluster: cluster}
}
