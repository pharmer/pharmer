package vfs

import (
	"context"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	stringz "github.com/appscode/go/strings"
	"github.com/appscode/go/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	_s3 "github.com/aws/aws-sdk-go/service/s3"
	"github.com/graymeta/stow"
	"github.com/graymeta/stow/azure"
	"github.com/graymeta/stow/google"
	"github.com/graymeta/stow/local"
	"github.com/graymeta/stow/s3"
	"github.com/graymeta/stow/swift"
	api "github.com/pharmer/pharmer/apis/v1alpha1"
	"github.com/pharmer/pharmer/credential"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
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
		} else if cfg.Store.S3 != nil {
			cred, err := cfg.GetCredential(cfg.Store.CredentialName)
			if err != nil {
				return nil, err
			}
			stowCfg := stow.ConfigMap{}

			keyID, foundKeyID := cred.Spec.Data[credential.AWSAccessKeyID]
			key, foundKey := cred.Spec.Data[credential.AWSSecretAccessKey]
			if foundKey && foundKeyID {
				stowCfg[s3.ConfigAccessKeyID] = string(keyID)
				stowCfg[s3.ConfigSecretKey] = string(key)
				stowCfg[s3.ConfigAuthType] = "accesskey"
			} else {
				stowCfg[s3.ConfigAuthType] = "iam"
			}
			if strings.HasSuffix(cfg.Store.S3.Endpoint, ".amazonaws.com") {
				// find region
				var sess *session.Session
				var err error
				if stowCfg[s3.ConfigAuthType] == "iam" {
					sess, err = session.NewSessionWithOptions(session.Options{
						Config: *aws.NewConfig(),
						// Support MFA when authing using assumed roles.
						SharedConfigState:       session.SharedConfigEnable,
						AssumeRoleTokenProvider: stscreds.StdinTokenProvider,
					})
				} else {
					config := &aws.Config{
						Credentials: credentials.NewStaticCredentials(string(keyID), string(key), ""),
						Region:      aws.String("us-east-1"),
					}
					sess, err = session.NewSessionWithOptions(session.Options{
						Config: *config,
						// Support MFA when authing using assumed roles.
						SharedConfigState:       session.SharedConfigEnable,
						AssumeRoleTokenProvider: stscreds.StdinTokenProvider,
					})
				}
				if err != nil {
					return nil, err
				}
				svc := _s3.New(sess)
				out, err := svc.GetBucketLocation(&_s3.GetBucketLocationInput{
					Bucket: types.StringP(cfg.Store.S3.Bucket),
				})
				if err != nil {
					return nil, err
				}
				stowCfg[s3.ConfigRegion] = stringz.Val(types.String(out.LocationConstraint), "us-east-1")
			} else {
				stowCfg[s3.ConfigEndpoint] = cfg.Store.S3.Endpoint
				if u, err := url.Parse(cfg.Store.S3.Endpoint); err == nil {
					stowCfg[s3.ConfigDisableSSL] = strconv.FormatBool(u.Scheme == "http")
				}
			}

			loc, err := stow.Dial(s3.Kind, stowCfg)
			if err != nil {
				return nil, errors.Errorf("failed to connect to S3 storage. Reason: %v", err)
			}
			name := cfg.Store.S3.Bucket
			container, err := loc.Container(name)
			if err != nil {
				container, err = loc.CreateContainer(name)
				if err != nil {
					return nil, errors.Errorf("failed to open storage container `%s`. Reason: %v", name, err)
				}
			}
			return New(container, cfg.Store.S3.Prefix), nil
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
				return nil, errors.Errorf("failed to connect to GCS storage. Reason: %v", err)
			}
			container, err := loc.Container(cfg.Store.GCS.Bucket)
			if err != nil {
				return nil, errors.Errorf("failed to open storage container `%s`. Reason: %v", cfg.Store.GCS.Bucket, err)
			}
			return New(container, cfg.Store.GCS.Prefix), nil
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
				return nil, errors.Errorf("failed to connect to Azure storage. Reason: %v", err)
			}
			name := cfg.Store.Azure.Container
			container, err := loc.Container(name)
			if err != nil {
				container, err = loc.CreateContainer(name)
				if err != nil {
					return nil, errors.Errorf("failed to open storage container `%s`. Reason: %v", name, err)
				}
			}
			return New(container, cfg.Store.Azure.Prefix), nil
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
				return nil, errors.Errorf("failed to connect to Swift storage. Reason: %v", err)
			}
			name := cfg.Store.Swift.Container
			container, err := loc.Container(name)
			if err != nil {
				container, err = loc.CreateContainer(name)
				if err != nil {
					return nil, errors.Errorf("failed to open storage container `%s`. Reason: %v", name, err)
				}
			}
			return New(container, cfg.Store.Swift.Prefix), nil
		}
		return nil, errors.New("missing store configuration")
	})
}

type FileStore struct {
	container stow.Container
	prefix    string
	owner     string
}

var _ store.Interface = &FileStore{}

func New(container stow.Container, prefix string) store.Interface {
	return &FileStore{container: container, prefix: prefix}
}

func (s *FileStore) Owner(id string) store.ResourceInterface {
	ret := *s
	ret.owner = id
	return &ret
}

func (s *FileStore) Credentials() store.CredentialStore {
	return &credentialFileStore{container: s.container, prefix: s.prefix, owner: s.owner}
}

func (s *FileStore) Clusters() store.ClusterStore {
	return &clusterFileStore{container: s.container, prefix: s.prefix, owner: s.owner}
}

func (s *FileStore) NodeGroups(cluster string) store.NodeGroupStore {
	return &nodeGroupFileStore{container: s.container, prefix: s.prefix, cluster: cluster, owner: s.owner}
}

func (s *FileStore) Certificates(cluster string) store.CertificateStore {
	return &certificateFileStore{container: s.container, prefix: s.prefix, cluster: cluster, owner: s.owner}
}

func (s *FileStore) SSHKeys(cluster string) store.SSHKeyStore {
	return &sshKeyFileStore{container: s.container, prefix: s.prefix, cluster: cluster, owner: s.owner}
}
