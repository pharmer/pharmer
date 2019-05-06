package credential

import (
	"encoding/json"
	"io/ioutil"

	"github.com/pharmer/cloud/pkg/apis"
	v1 "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GCE struct {
	CommonSpec

	credentialFile string
	projectID      string
}

func NewGCE() *GCE {
	return &GCE{}
}

func (c GCE) ProjectID() string { return get(c.Data, GCEProjectID, c.projectID) }
func (c GCE) ServiceAccount() string {
	if v, ok := c.Data[GCEServiceAccount]; ok {
		return v
	}

	data, err := ioutil.ReadFile(c.credentialFile)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func (c *GCE) LoadFromEnv() {
	c.CommonSpec.LoadFromEnv(c.Format())
}

func (c GCE) IsValid() (bool, error) {
	return c.CommonSpec.IsValid(c.Format())
}

func (c *GCE) Load(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	d := make(map[string]string)
	if err = json.Unmarshal(data, &d); err != nil {
		return err
	}

	c.Data = make(map[string]string)
	c.Data[GCEServiceAccount] = string(data)
	c.Data[GCEProjectID] = d["project_id"]
	return nil
}

func (c *GCE) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&c.credentialFile, "gce.credential_file", "c", c.credentialFile, "Location of cloud credential file (required when --provider=gce)")
	fs.StringVar(&c.projectID, "gce.project_id", c.projectID, "provide this flag when provider is gce")
}

func (_ GCE) RequiredFlags() []string {
	return []string{
		"gce.credential_file",
		"gce.project_id",
	}
}

func (_ GCE) Format() v1.CredentialFormat {
	return v1.CredentialFormat{
		ObjectMeta: metav1.ObjectMeta{
			Name: apis.GCE,
			Labels: map[string]string{
				apis.KeyCloudProvider: apis.GCE,
			},
			Annotations: map[string]string{
				apis.KeyClusterCredential: "",
				apis.KeyDNSCredential:     "",
				apis.KeyStorageCredential: "",
			},
		},
		Spec: v1.CredentialFormatSpec{
			Provider:      apis.GCE,
			DisplayFormat: "json",
			Fields: []v1.CredentialField{
				{
					Envconfig: "GCE_PROJECT_ID",
					Form:      "gce_project_id",
					JSON:      GCEProjectID,
					Label:     "Google Cloud Project ID",
					Input:     "text",
				},
				{
					Envconfig: "GCE_SERVICE_ACCOUNT",
					Form:      "gce_service_account",
					JSON:      GCEServiceAccount,
					Label:     "Google Cloud Service Account",
					Input:     "textarea",
				},
			},
		},
	}
}
