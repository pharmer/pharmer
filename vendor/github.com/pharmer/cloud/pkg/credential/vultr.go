package credential

import (
	"github.com/pharmer/cloud/pkg/apis"
	v1 "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Vultr struct {
	CommonSpec

	token string
}

func (c Vultr) Token() string { return get(c.Data, VultrAPIToken, c.token) }

func (c *Vultr) LoadFromEnv() {
	c.CommonSpec.LoadFromEnv(c.Format())
}

func (c Vultr) IsValid() (bool, error) {
	return c.CommonSpec.IsValid(c.Format())
}

func (c *Vultr) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.token, apis.Vultr+"."+VultrAPIToken, c.token, "Vultr token")
}

func (_ Vultr) RequiredFlags() []string {
	return []string{apis.Vultr + "." + VultrAPIToken}
}

func (_ Vultr) Format() v1.CredentialFormat {
	return v1.CredentialFormat{
		ObjectMeta: metav1.ObjectMeta{
			Name: apis.Vultr,
			Labels: map[string]string{
				apis.KeyCloudProvider: apis.Vultr,
			},
			Annotations: map[string]string{
				apis.KeyClusterCredential: "",
				apis.KeyDNSCredential:     "",
			},
		},
		Spec: v1.CredentialFormatSpec{
			Provider:      apis.Vultr,
			DisplayFormat: "field",
			Fields: []v1.CredentialField{
				{
					Envconfig: "VULTR_TOKEN",
					Form:      "vultr_token",
					JSON:      VultrAPIToken,
					Label:     "Personal Access Token",
					Input:     "password",
				},
			},
		},
	}
}
