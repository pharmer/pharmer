package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	yc "github.com/appscode/go/encoding/yaml"
	"github.com/appscode/pharmer/api"
	"github.com/ghodss/yaml"
	"k8s.io/client-go/util/homedir"
)

type LocalSpec struct {
	Path string `json:"path,omitempty"`
}

type S3Spec struct {
	Endpoint string `json:"endpoint,omitempty"`
	Bucket   string `json:"bucket,omiempty"`
	Prefix   string `json:"prefix,omitempty"`
}

type GCSSpec struct {
	Bucket string `json:"bucket,omiempty"`
	Prefix string `json:"prefix,omitempty"`
}

type AzureSpec struct {
	Container string `json:"container,omitempty"`
	Prefix    string `json:"prefix,omitempty"`
}

type SwiftSpec struct {
	Container string `json:"container,omitempty"`
	Prefix    string `json:"prefix,omitempty"`
}

type StorageBackend struct {
	CredentialName string `json:"credentialName,omitempty"`

	Local *LocalSpec `json:"local,omitempty"`
	S3    *S3Spec    `json:"s3,omitempty"`
	GCS   *GCSSpec   `json:"gcs,omitempty"`
	Azure *AzureSpec `json:"azure,omitempty"`
	Swift *SwiftSpec `json:"swift,omitempty"`
}

type DNSProvider struct {
	CredentialName string `json:"credentialName,omitempty"`
}

type PharmerConfig struct {
	api.TypeMeta `json:",inline,omitempty"`
	Context      string           `json:context,omitempty`
	Credentials  []api.Credential `json:"credentials,omitempty"`
	Store        StorageBackend   `json:"store,omitempty"`
	DNS          *DNSProvider     `json:"dns,omitempty"`
}

func LoadConfig(configPath string) (*PharmerConfig, error) {
	if _, err := os.Stat(configPath); err != nil {
		return nil, err
	}
	os.Chmod(configPath, 0600)

	config := &PharmerConfig{}
	bytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	jsonData, err := yc.ToJSON(bytes)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(jsonData, config)
	return config, err
}

func (pc PharmerConfig) Save(configPath string) error {
	data, err := yaml.Marshal(pc)
	if err != nil {
		return err
	}
	os.MkdirAll(filepath.Dir(configPath), 0755)
	return ioutil.WriteFile(configPath, data, 0600)
}

func (pc PharmerConfig) GetStoreType() string {
	if pc.Store.Local != nil {
		return "Local"
	} else if pc.Store.S3 != nil {
		return "S3"
	} else if pc.Store.S3 != nil {
		return "S3"
	} else if pc.Store.GCS != nil {
		return "GCS"
	} else if pc.Store.Azure != nil {
		return "Azure"
	} else if pc.Store.Swift != nil {
		return "OpenStack Swift"
	}
	return "<Unknown>"
}

func (pc PharmerConfig) GetDNSProviderType() string {
	if pc.DNS == nil {
		return "-"
	}
	if pc.DNS.CredentialName == "" {
		return "-"
	}
	for _, c := range pc.Credentials {
		if c.Name == pc.DNS.CredentialName {
			return c.Spec.Provider
		}
	}
	return "<Unknown>"
}

func NewLocalConfig() *PharmerConfig {
	return &PharmerConfig{
		TypeMeta: api.TypeMeta{
			Kind: "PharmerConfig",
		},
		Context: "default",
		Store: StorageBackend{
			Local: &LocalSpec{
				Path: filepath.Join(homedir.HomeDir(), ".pharmer", "store.d"),
			},
		},
	}
}

func CreateDefaultConfigIfAbsent() error {
	configPath := DefaultConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return NewLocalConfig().Save(configPath)
	}
	return nil
}

func DefaultConfigPath() string {
	return filepath.Join(homedir.HomeDir(), ".pharmer", "config.d", "default")
}

func ConfigDir() string {
	return filepath.Join(homedir.HomeDir(), ".pharmer", "config.d")
}
