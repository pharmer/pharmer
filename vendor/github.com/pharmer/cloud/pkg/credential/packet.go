package credential

import (
	"github.com/pharmer/cloud/pkg/apis"
	v1 "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Packet struct {
	CommonSpec

	token     string
	projectID string
}

func (c Packet) APIKey() string    { return get(c.Data, PacketAPIKey, c.token) }
func (c Packet) ProjectID() string { return get(c.Data, PacketProjectID, c.projectID) }

func (c *Packet) LoadFromEnv() {
	c.CommonSpec.LoadFromEnv(c.Format())
}

func (c Packet) IsValid() (bool, error) {
	return c.CommonSpec.IsValid(c.Format())
}

func (c *Packet) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.token, apis.Packet+"."+PacketAPIKey, c.token, "Packet api key")
	fs.StringVar(&c.projectID, apis.Packet+"."+PacketProjectID, c.projectID, "Packet project id")
}

func (_ Packet) RequiredFlags() []string {
	return []string{apis.Packet + "." + PacketAPIKey}
}

func (_ Packet) Format() v1.CredentialFormat {
	return v1.CredentialFormat{
		ObjectMeta: metav1.ObjectMeta{
			Name: apis.Packet,
			Annotations: map[string]string{
				apis.KeyClusterCredential: "",
			},
		},
		Spec: v1.CredentialFormatSpec{
			Provider:      apis.Packet,
			DisplayFormat: "field",
			Fields: []v1.CredentialField{
				{
					Envconfig: "PACKET_PROJECT_ID",
					Form:      "packet_project_id",
					JSON:      "projectID",
					Label:     "Project Id",
					Input:     "text",
				},
				{
					Envconfig: "PACKET_API_KEY",
					Form:      "packet_api_key",
					JSON:      "apiKey",
					Label:     "API Key",
					Input:     "password",
				},
			},
		},
	}
}
