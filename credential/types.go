package credential

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/data/files"
)

const (
	AWSAccessKeyID       = "accessKeyID"
	AWSSecretAccessKey   = "secretAccessKey"
	AzureClientID        = "clientID"
	AzureClientSecret    = "clientSecret"
	AzureStorageAccount  = "account"
	AzureStorageKey      = "key"
	AzureSubscriptionID  = "subscriptionID"
	AzureTenantID        = "tenantID"
	DigitalOceanToken    = "token"
	GCEServiceAccount    = "serviceAccount"
	GCEProjectID         = "projectID"
	HertznerPassword     = "password"
	HertznerUsername     = "username"
	LinodeAPIToken       = "token"
	PacketAPIKey         = "apiKey"
	PacketProjectID      = "projectID"
	ScalewayOrganization = "organization"
	ScalewayToken        = "token"
	SoftlayerAPIKey      = "apiKey"
	SoftlayerUsername    = "username"
	VultrAPIToken        = "token"
)

type generic api.CredentialSpec

func (c generic) Load(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	if c.Data != nil {
		c.Data = map[string]string{}
	}
	return json.Unmarshal(data, &c.Data)
}

func (c generic) IsValid() bool {
	if cf, ok := files.GetCredentialFormat(c.Provider); ok {
		for _, f := range cf.Fields {
			if _, found := c.Data[f.JSON]; !found {
				return false
			}
		}
	}
	return true
}

func (c generic) ToRawMap() map[string]string {
	result := map[string]string{}
	for k, v := range c.Data {
		result[k] = v
	}
	return result
}

func (c generic) ToMaskedMap() map[string]string {
	result := map[string]string{}
	if cf, ok := files.GetCredentialFormat(c.Provider); ok {
		for _, f := range cf.Fields {
			if f.Input == "password" {
				// TODO: FixIt! mask it
				result[f.JSON] = "*****"
			} else {
				result[f.JSON] = c.Data[f.JSON]
			}
		}
	}
	return result
}

func (c *generic) LoadFromEnv() {
	if c.Data == nil {
		c.Data = map[string]string{}
	}
	if cf, ok := files.GetCredentialFormat(c.Provider); ok {
		for _, f := range cf.Fields {
			c.Data[f.JSON] = os.Getenv(f.Envconfig)
		}
	}
}
