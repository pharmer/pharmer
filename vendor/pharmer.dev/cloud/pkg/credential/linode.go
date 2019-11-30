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

type Linode struct {
	CommonSpec

	token string
}

func (c Linode) APIToken() string { return get(c.Data, LinodeAPIToken, c.token) }

func (c *Linode) LoadFromEnv() {
	c.CommonSpec.LoadFromEnv(c.Format())
}

func (c Linode) IsValid() (bool, error) {
	return c.CommonSpec.IsValid(c.Format())
}

func (c *Linode) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.token, apis.Linode+"."+LinodeAPIToken, c.token, "Linode api token")
}

func (_ Linode) RequiredFlags() []string {
	return []string{apis.Linode + "." + LinodeAPIToken}
}

func (_ Linode) Format() v1.CredentialFormat {
	return v1.CredentialFormat{
		ObjectMeta: metav1.ObjectMeta{
			Name: apis.Linode,
			Annotations: map[string]string{
				apis.KeyClusterCredential: "",
				apis.KeyDNSCredential:     "",
			},
		},
		Spec: v1.CredentialFormatSpec{
			Provider:      apis.Linode,
			DisplayFormat: "field",
			Fields: []v1.CredentialField{
				{
					Envconfig: "LINODE_TOKEN",
					Form:      "linode_token",
					JSON:      LinodeAPIToken,
					Label:     "Token",
					Input:     "password",
				},
			},
		},
	}
}
