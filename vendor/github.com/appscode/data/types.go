package data

import (
	"encoding/json"
	"math/big"
	"time"
)

type Product struct {
	Sku              string          `json:"sku"`
	Type             string          `json:"type"`
	SubType          string          `json:"sub_type"`
	QuotaRequired    string          `json:"quota_required"`
	DisplayName      string          `json:"display_name"`
	PricingModel     string          `json:"pricing_model"`
	SubscriptionType string          `json:"subscription_type"`
	TimeUnit         string          `json:"time_unit"`
	PricingFormula   string          `json:"pricing_formula"`
	Metadata         json.RawMessage `json:"metadata"`
	Quota            json.RawMessage `json:"quota"`
	DateStarted      time.Time       `json:"date_started"`
	DateEnded        time.Time       `json:"date_ended"`
}

type ClusterData struct {
	Kubernetes []*Product `json:"kubernetes"`
}

type PkgData struct {
	Package []*Product `json:"pkg"`
}

type Package struct {
	Sku              string `json:"sku"`
	Type             string `json:"type"`
	DisplayName      string `json:"display_name"`
	PricingModel     string `json:"pricing_model"`
	SubscriptionType string `json:"subscription_type"`
	TimeUnit         string `json:"time_unit"`
	PricingFormula   string `json:"pricing_formula"`
	Quota            struct {
		PkgUser           int `json:"pkg.user"`
		PhabricatorDiskGB int `json:"phabricator.disk_gb"`
		ArtifactDiskGB    int `json:"artifact.disk_gb"`
	} `json:"metadata"`
	DateStarted time.Time `json:"date_started"`
	DateEnded   time.Time `json:"date_ended"`
}

type PackageProduct struct {
	Package []*Package `json:"package"`
}

type Region struct {
	Location string   `json:"location"`
	Region   string   `json:"region"`
	Zones    []string `json:"zones,omitempty"`
}

type InstanceType struct {
	ExternalSku string      `json:"external_sku"`
	Description string      `json:"description"`
	Category    string      `json:"category"`
	CPU         int         `json:"cpu"`
	RAM         interface{} `json:"ram"`
	Disk        int         `json:"disk"`
	Regions     []string    `json:"regions,omitempty"`
}

type CloudKubernetes struct {
	DefaultSetup struct {
		NodeGroup struct {
			M4Large int `json:"m4.large"`
		} `json:"nodeGroup"`
	} `json:"default_setup"`
	VersionsByEnv map[string][]*CloudKubernetesVersion `json:"versions_by_env"`
}

type CloudKubernetesVersion struct {
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Apps        map[string]string `json:"apps"`
	Deprecated  bool              `json:"deprecated"`
}

type CloudProvider struct {
	Name          string           `json:"name"`
	Envs          []string         `json:"envs"`
	Regions       []*Region        `json:"regions"`
	InstanceTypes []*InstanceType  `json:"instance_types"`
	Kubernetes    *CloudKubernetes `json:"kubernetes"`
}

type ClusterProviders struct {
	Provider map[string]CloudProvider `json:"cloud_provider"`
}

type DNSField struct {
	Envconfig string `json:"envconfig"`
	JSON      string `json:"json"`
	Name      string `json:"name"`
	Label     string `json:"label"`
}
type DNSProviders struct {
	Provider    string      `json:"provider"`
	InputFormat string      `json:"input_format"`
	Fields      []*DNSField `json:"fields"`
}

type Money string

const moneyPrecision = 40

func (m Money) Float() (*big.Float, bool) {
	return new(big.Float).SetPrec(moneyPrecision).SetString(string(m))
}

func (m Money) String() string {
	return string(m)
}
