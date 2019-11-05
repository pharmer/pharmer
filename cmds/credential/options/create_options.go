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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type CredentialCreateConfig struct {
	Name     string
	Provider string
	FromEnv  bool
	FromFile string
	Issue    bool
}

func NewCredentialCreateConfig() *CredentialCreateConfig {
	return &CredentialCreateConfig{
		Provider: "",
		FromEnv:  false,
		FromFile: "",
		Issue:    false,
	}
}

func (c *CredentialCreateConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&c.Provider, "provider", "p", c.Provider, "Name of the Cloud provider")
	fs.BoolVarP(&c.FromEnv, "from-env", "l", c.FromEnv, "Load credential data from ENV.")
	fs.StringVarP(&c.FromFile, "from-file", "f", c.FromFile, "Load credential data from file")
	fs.BoolVar(&c.Issue, "issue", c.Issue, "Issue credential")
}

func (c *CredentialCreateConfig) ValidateFlags(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("missing credential name")
	}
	if len(args) > 1 {
		return errors.New("multiple credential name provided")
	}
	c.Name = args[0]
	return nil
}
