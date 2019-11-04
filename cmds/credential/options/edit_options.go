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

type CredentialEditConfig struct {
	Name        string
	File        string
	DoNotDelete bool
	Output      string
	Owner       string
}

func NewCredentialEditConfig() *CredentialEditConfig {
	return &CredentialEditConfig{
		File:        "",
		DoNotDelete: false,
		Output:      "yaml",
	}
}

func (c *CredentialEditConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringP("file", "f", "", "Load credential data from file")
	fs.BoolP("do-not-delete", "", false, "Set do not delete flag")
	fs.StringP("output", "o", "yaml", "Output format. One of: yaml|json.")
	fs.StringVarP(&c.Owner, "owner", "", c.Owner, "Current user id")
}

func (c *CredentialEditConfig) ValidateFlags(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("missing credential name")
	}
	if len(args) > 1 {
		return errors.New("multiple credential name provided")
	}
	c.Name = args[0]
	return nil
}
