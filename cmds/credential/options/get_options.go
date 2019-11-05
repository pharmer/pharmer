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
package options

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type CredentialGetConfig struct {
	Credentials []string
	Output      string
	Owner       string
}

func NewCredentialGetConfig() *CredentialGetConfig {
	return &CredentialGetConfig{
		Output: "",
	}
}

func (c *CredentialGetConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&c.Output, "output", "o", c.Output, "Output format. One of: json|yaml|wide")
	fs.StringVarP(&c.Owner, "owner", "", c.Owner, "Current user id")
}

func (c *CredentialGetConfig) ValidateFlags(cmd *cobra.Command, args []string) error {
	c.Credentials = args
	return nil
}
