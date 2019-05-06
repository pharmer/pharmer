package credential

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/pharmer/cloud/pkg/apis"
	api "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	v1 "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	AWSRegion            = "region"
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

func (c *CommonSpec) LoadFromEnv(cf v1.CredentialFormat) {
	if c.Data == nil {
		c.Data = map[string]string{}
	}
	for _, f := range cf.Spec.Fields {
		c.Data[f.JSON] = os.Getenv(f.Envconfig)
	}
}

func (c CommonSpec) IsValid(cf v1.CredentialFormat) (bool, error) {
	for _, f := range cf.Spec.Fields {
		if _, found := c.Data[f.JSON]; !found {
			return false, errors.Errorf("missing key: %s", f.JSON)
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
	bl := sets.NewString("secret", "token", "password", "credential")
	result := c.ToRawMap()
	for _, keyword := range bl.UnsortedList() {
		if _, ok := result[keyword]; ok {
			result[keyword] = "***REDACTED***"
		}
	}
	for k, v := range result {
		if len(v) > 10 {
			// TODO: FixIt! show shorter version of large amount of data
			result[k] = "<data>"
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
	case apis.GCE:
		gce := NewGCE()
		if err := gce.Load(fileName); err != nil {
			return CommonSpec{}, err
		}
		return gce.CommonSpec, nil
	case apis.AWS:
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

func GetFormat(provider string) v1.CredentialFormat {
	switch provider {
	case apis.GCE:
		return GCE{}.Format()
	case apis.DigitalOcean:
		return DigitalOcean{}.Format()
	case apis.Packet:
		return Packet{}.Format()
	case apis.AWS:
		return AWS{}.Format()
	case apis.Azure:
		return Azure{}.Format()
	case apis.AzureStorage:
		return AzureStorage{}.Format()
	case apis.Vultr:
		return Vultr{}.Format()
	case apis.Linode:
		return Linode{}.Format()
	case apis.Scaleway:
		return Scaleway{}.Format()
	}
	panic("unknown provider " + provider)
}

func get(m map[string]string, k, alt string) string {
	if v, ok := m[k]; ok {
		return v
	}
	return alt
}
