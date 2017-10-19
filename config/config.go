package config

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/appscode/go-dns"
	dns_provider "github.com/appscode/go-dns/provider"
	yc "github.com/appscode/go/encoding/yaml"
	_env "github.com/appscode/go/env"
	"github.com/appscode/go/log"
	api "github.com/appscode/pharmer/apis/v1alpha1"
	"github.com/appscode/pharmer/store"
	"github.com/appscode/pharmer/store/providers/fake"
	"github.com/appscode/pharmer/store/providers/vfs"
	"github.com/ghodss/yaml"
	flag "github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/homedir"
)

func LoadConfig(configPath string) (*api.PharmerConfig, error) {
	if _, err := os.Stat(configPath); err != nil {
		return nil, err
	}
	os.Chmod(configPath, 0600)

	config := &api.PharmerConfig{}
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

func Save(pc *api.PharmerConfig, configPath string) error {
	data, err := yaml.Marshal(pc)
	if err != nil {
		return err
	}
	os.MkdirAll(filepath.Dir(configPath), 0755)
	return ioutil.WriteFile(configPath, data, 0600)
}

func AddFlags(fs *flag.FlagSet) {
	fs.String("config-file", "", "Path to Pharmer config file")
	// TODO: change env to Prod
	fs.String("env", _env.Dev.String(), "Environment used to enable debugging")
}

func GetConfigFile(fs *flag.FlagSet) (string, bool) {
	cfgFile, err := fs.GetString("config-file")
	if err != nil {
		log.Fatalf("can't accessing flag `config-file`. Reason: %v", err)
	}
	if cfgFile == "" {
		return filepath.Join(homedir.HomeDir(), ".pharmer", "config.d", "default"), false
	}
	return cfgFile, true
}

func GetEnv(fs *flag.FlagSet) _env.Environment {
	e, err := fs.GetString("env")
	if err != nil {
		log.Fatalf("can't accessing flag `config-file`. Reason: %v", err)
	}
	return _env.FromString(e)
}

func ConfigDir(fs *flag.FlagSet) string {
	cfgFile, _ := GetConfigFile(fs)
	return filepath.Dir(cfgFile)
}

func NewDefaultConfig() *api.PharmerConfig {
	return &api.PharmerConfig{
		TypeMeta: metav1.TypeMeta{
			Kind: "PharmerConfig",
		},
		Context: "default",
		Store: api.StorageBackend{
			Local: &api.LocalSpec{
				Path: filepath.Join(homedir.HomeDir(), ".pharmer", "store.d"),
			},
		},
	}
}

func NewStoreProvider(ctx context.Context, cfg *api.PharmerConfig) store.Interface {
	if store, err := store.GetProvider(vfs.UID, ctx, cfg); err == nil {
		return store
	}
	return &fake.FakeStore{}
}

func NewDNSProvider(cfg *api.PharmerConfig) dns_provider.Provider {
	if cfg.DNS != nil {
		if cred, err := cfg.GetCredential(cfg.DNS.CredentialName); err == nil {
			if dp, err := dns.Default(cred.Spec.Provider); err == nil {
				return dp
			}
		}
	}
	return &api.FakeDNSProvider{}
}
