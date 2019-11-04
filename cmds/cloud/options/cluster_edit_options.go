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

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type ClusterEditConfig struct {
	ClusterName       string
	File              string
	KubernetesVersion string
	Locked            bool
	Output            string
}

func NewClusterEditConfig() *ClusterEditConfig {
	return &ClusterEditConfig{
		ClusterName:       "",
		File:              "",
		KubernetesVersion: "",
		Locked:            false,
		Output:            "yaml",
	}
}

func (c *ClusterEditConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&c.File, "file", "f", c.File, "Load cluster data from file")
	//TODO: Add necessary flags that will be used for update
	fs.StringVar(&c.KubernetesVersion, "kubernetes-version", c.KubernetesVersion, "Kubernetes version")
	fs.BoolVar(&c.Locked, "locked", c.Locked, "If true, locks cluster from deletion")
	fs.StringVarP(&c.Output, "output", "o", c.Output, "Output format. One of: yaml|json.")

}

func (c *ClusterEditConfig) ValidateFlags(cmd *cobra.Command, args []string) error {
	if cmd.Flags().Changed("file") {
		if len(args) != 0 {
			return errors.New("no argument can be provided when --file flag is used")
		}
	}
	if len(args) == 0 {
		return errors.New("missing cluster name")
	}
	if len(args) > 1 {
		return errors.New("multiple cluster name provided")
	}
	c.ClusterName = strings.ToLower(args[0])
	return nil
}

func (c *ClusterEditConfig) CheckForUpdateFlags() bool {
	if c.Locked || c.KubernetesVersion != "" {
		return true
	}
	return false
}
