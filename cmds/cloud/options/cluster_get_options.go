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
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type ClusterGetConfig struct {
	Clusters []string
	Output   string
}

func NewClusterGetConfig() *ClusterGetConfig {
	return &ClusterGetConfig{
		Output: "",
	}
}

func (c *ClusterGetConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&c.Output, "output", "o", c.Output, "Output format. One of: json|yaml|wide")
}

func (c *ClusterGetConfig) ValidateFlags(cmd *cobra.Command, args []string) error {
	c.Clusters = func(names []string) []string {
		for i := range names {
			names[i] = strings.ToLower(names[i])
		}
		return names
	}(args)
	return nil
}
