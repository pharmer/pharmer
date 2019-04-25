package credential

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/data/files"
	"github.com/pkg/errors"
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
	SwiftUsername        = "username"
	SwiftKey             = "key"
	SwiftTenantName      = "tenantName"
	SwiftTenantAuthURL   = "tenantAuthURL"
	SwiftDomain          = "domain"
	SwiftRegion          = "region"
	SwiftTenantId        = "tenantID"
	SwiftTenantDomain    = "tenantDomain"
	SwiftTrustId         = "trustID"
	SwiftStorageURL      = "storageURL"
	SwiftAuthToken       = "authToken"
	VultrAPIToken        = "token"
	OvhUsername          = "username"
	OvhPassword          = "password"
	OvhTenantID          = "tenantID"
)

type CommonSpec api.CredentialSpec

func (c *CommonSpec) Load(filename string) error {
	return c.LoadFromJSON(filename)
}

func (c *CommonSpec) LoadFromJSON(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	if c.Data != nil {
		c.Data = map[string]string{}
	}
	return json.Unmarshal(data, &c.Data)
}

func (c *CommonSpec) LoadFromEnv() {
	if c.Data == nil {
		c.Data = map[string]string{}
	}
	if cf, ok := files.GetCredentialFormat(c.Provider); ok {
		for _, f := range cf.Fields {
			c.Data[f.JSON] = os.Getenv(f.Envconfig)
		}
	}
}

func (c CommonSpec) IsValid() (bool, error) {
	if cf, ok := files.GetCredentialFormat(c.Provider); ok {
		for _, f := range cf.Fields {
			if _, found := c.Data[f.JSON]; !found {
				return false, errors.Errorf("missing key: %s", f.JSON)
			}
		}
	}
	return true, nil
}

func (c CommonSpec) ToRawMap() map[string]string {
	result := map[string]string{}
	for k, v := range c.Data {
		result[k] = v
	}
	return result
}

func (c CommonSpec) ToMaskedMap() map[string]string {
	result := map[string]string{}
	if cf, ok := files.GetCredentialFormat(c.Provider); ok {
		for _, f := range cf.Fields {
			if f.Input == "password" {
				// TODO: FixIt! mask it
				result[f.JSON] = "*****"
			} else {
				if len(c.Data[f.JSON]) > 50 {
					// TODO: FixIt! show shorter version of large amount of data
					result[f.JSON] = "<data>"
				} else {
					result[f.JSON] = c.Data[f.JSON]
				}

			}
		}
	}
	return result
}

func (c CommonSpec) String() string {
	var buf bytes.Buffer
	for k, v := range c.ToMaskedMap() {
		if buf.Len() > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(k)
		buf.WriteString("=")
		buf.WriteString(v)
	}
	return buf.String()
}

func LoadCredentialDataFromJson(provider string, fileName string) (CommonSpec, error) {
	switch provider {
	case "GoogleCloud":
		gce := NewGCE()
		if err := gce.Load(fileName); err != nil {
			return CommonSpec{}, err
		}
		return gce.CommonSpec, nil
	case "AWS":
		aws := NewAWS()
		if err := aws.Load(fileName); err != nil {
			return CommonSpec{}, err
		}
		return aws.CommonSpec, nil
	default:
		commonSpec := CommonSpec{}
		if err := commonSpec.Load(fileName); err != nil {
			return CommonSpec{}, err
		}
		return commonSpec, nil
	}
}
