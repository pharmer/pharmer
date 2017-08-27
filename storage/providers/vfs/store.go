package vfs

import (
	"context"
	"errors"
	"net/url"
	"strconv"

	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/credential"
	"github.com/appscode/pharmer/storage"
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
	storage.RegisterProvider(UID, func(ctx context.Context, cfg *api.PharmerConfig) (storage.Interface, error) {
		if cfg.Store.Local != nil {
			stowCfg := stow.ConfigMap{
				local.ConfigKeyPath: cfg.Store.Local.Path,
			}
			loc, err := stow.Dial(local.Kind, stowCfg)
			if err != nil {
				return nil, err
			}
			container, err := loc.Container("xyz")
			if err != nil {
				return nil, err
			}
			return &FileStore{container: container}, nil
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
				return nil, err
			}
			container, err := loc.Container(cfg.Store.S3.Bucket)
			if err != nil {
				return nil, err
			}
			return &FileStore{container: container}, nil
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
				return nil, err
			}
			container, err := loc.Container(cfg.Store.GCS.Bucket)
			if err != nil {
				return nil, err
			}
			return &FileStore{container: container}, nil
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
				return nil, err
			}
			container, err := loc.Container(cfg.Store.Azure.Container)
			if err != nil {
				return nil, err
			}
			return &FileStore{container: container}, nil
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
				return nil, err
			}
			container, err := loc.Container(cfg.Store.Swift.Container)
			if err != nil {
				return nil, err
			}
			return &FileStore{container: container}, nil
		}
		return nil, errors.New("Missing store configuration")
	})
}

type FileStore struct {
	container stow.Container
}

var _ storage.Interface = &FileStore{}

func (s *FileStore) Credentials() storage.CredentialStore {
	return &CredentialFileStore{container: s.container}
}

func (s *FileStore) Clusters() storage.ClusterStore {
	return &ClusterFileStore{container: s.container}
}

func (s *FileStore) Instances(name string) storage.InstanceStore {
	return &InstanceFileStore{container: s.container, cluster: name}
}

func (s *FileStore) Certificates(name string) storage.CertificateStore {
	return &CertificateFileStore{container: s.container, cluster: name}
}

func (s *FileStore) SSHKeys(name string) storage.SSHKeyStore {
	return &SSHKeyFileStore{container: s.container, cluster: name}
}
