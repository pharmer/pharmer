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

func NewLocalConfig() *api.PharmerConfig {
	return &api.PharmerConfig{
		TypeMeta: api.TypeMeta{
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

func CreateDefaultConfigIfAbsent() error {
	configPath := DefaultConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return Save(NewLocalConfig(), configPath)
	}
	return nil
}

func DefaultConfigPath() string {
	return filepath.Join(homedir.HomeDir(), ".pharmer", "config.d", "default")
}

func ConfigDir() string {
	return filepath.Join(homedir.HomeDir(), ".pharmer", "config.d")
}
