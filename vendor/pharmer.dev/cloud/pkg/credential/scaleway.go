/*
Copyright The Pharmer Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package credential

import (
	"pharmer.dev/cloud/apis"
	v1 "pharmer.dev/cloud/apis/cloud/v1"

	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Scaleway struct {
	CommonSpec

	token        string
	organization string
}

func (c Scaleway) Organization() string { return get(c.Data, ScalewayOrganization, c.organization) }
func (c Scaleway) Token() string        { return get(c.Data, ScalewayToken, c.token) }

func (c *Scaleway) LoadFromEnv() {
	c.CommonSpec.LoadFromEnv(c.Format())
}

func (c Scaleway) IsValid() (bool, error) {
	return c.CommonSpec.IsValid(c.Format())
}

func (c *Scaleway) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.token, apis.Scaleway+"."+ScalewayToken, c.token, "Scaleway token")
	fs.StringVar(&c.organization, apis.Scaleway+"."+ScalewayOrganization, c.organization, "Scaleway organization")
}

func (_ Scaleway) RequiredFlags() []string {
	return []string{
		apis.Scaleway + "." + ScalewayToken,
		apis.Scaleway + "." + ScalewayOrganization,
	}
}

func (_ Scaleway) Format() v1.CredentialFormat {
	return v1.CredentialFormat{
		ObjectMeta: metav1.ObjectMeta{
			Name: apis.Scaleway,
			Labels: map[string]string{
				apis.KeyCloudProvider: apis.Scaleway,
			},
			Annotations: map[string]string{
				apis.KeyClusterCredential: "",
			},
		},
		Spec: v1.CredentialFormatSpec{
			Provider:      apis.Scaleway,
			DisplayFormat: "field",
			Fields: []v1.CredentialField{
				{
					Envconfig: "SCALEWAY_ORGANIZATION",
					Form:      "scaleway_organization",
					JSON:      ScalewayOrganization,
					Label:     "Organization",
					Input:     "text",
				},
				{
					Envconfig: "SCALEWAY_TOKEN",
					Form:      "scaleway_token",
					JSON:      ScalewayToken,
					Label:     "Token",
					Input:     "password",
				},
			},
		},
	}
}
