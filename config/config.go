package config

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/appscode/go-dns/aws"
	"github.com/appscode/go-dns/azure"
	"github.com/appscode/go-dns/cloudflare"
	"github.com/appscode/go-dns/digitalocean"
	"github.com/appscode/go-dns/googlecloud"
	"github.com/appscode/go-dns/linode"
	"github.com/appscode/go-dns/vultr"
	yc "github.com/appscode/go/encoding/yaml"
	"github.com/appscode/log"
	"github.com/ghodss/yaml"
	"github.com/graymeta/stow"
	"github.com/spf13/cobra"
)

type Context struct {
	Name     string         `json:"name"`
	Provider string         `json:"provider"`
	Config   stow.ConfigMap `json:"config"`
	DNS      struct {
		Provider     string               `json:"provider,omitempty"`
		AWS          aws.Options          `json:"aws,omitempty"`
		Azure        azure.Options        `json:"azure,omitempty"`
		Cloudflare   cloudflare.Options   `json:"cloudflare,omitempty"`
		Digitalocean digitalocean.Options `json:"digitalocean,omitempty"`
		Gcloud       googlecloud.Options  `json:"gcloud,omitempty"`
		Linode       linode.Options       `json:"linode,omitempty"`
		Vultr        vultr.Options        `json:"vultr,omitempty"`
	} `json:"dns"`
}

type PharmerConfig struct {
	Contexts       []*Context `json:"contexts"`
	CurrentContext string     `json:"current-context"`
}

func (pc PharmerConfig) Context(ctx string) *Context {
	name := pc.CurrentContext
	if ctx != "" {
		name = ctx
	}
	for _, c := range pc.Contexts {
		if c.Name == name {
			return c
		}
	}
	// TODO: FixIt! (set a Null Context)
	return nil
}

func GetConfigPath(cmd *cobra.Command) string {
	s, err := cmd.Flags().GetString("osmconfig")
	if err != nil {
		log.Fatalf("error accessing flag osmconfig for command %s: %v", cmd.Name(), err)
	}
	return s
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

func (pc *PharmerConfig) Save(configPath string) error {
	data, err := yaml.Marshal(pc)
	if err != nil {
		return err
	}
	os.MkdirAll(filepath.Dir(configPath), 0755)
	if err := ioutil.WriteFile(configPath, data, 0600); err != nil {
		return err
	}
	return nil
}

func (pc *PharmerConfig) Dial(cliCtx string) (stow.Location, error) {
	ctx := pc.CurrentContext
	if cliCtx != "" {
		ctx = cliCtx
	}
	for _, osmCtx := range pc.Contexts {
		if osmCtx.Name == ctx {
			return stow.Dial(osmCtx.Provider, osmCtx.Config)
		}
	}
	return nil, errors.New("Failed to determine context.")
}
