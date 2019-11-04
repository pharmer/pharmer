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

	"github.com/appscode/go/flags"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type NodeGroupEditConfig struct {
	ClusterName string
	NgName      string
	File        string
	DoNotDelete bool
	Output      string
}

func NewNodeGroupEditConfig() *NodeGroupEditConfig {
	return &NodeGroupEditConfig{
		ClusterName: "",
		NgName:      "",
		File:        "",
		DoNotDelete: false,
		Output:      "yaml",
	}
}

func (c *NodeGroupEditConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&c.ClusterName, "cluster", "k", c.ClusterName, "Name of the Kubernetes cluster")
	fs.StringVarP(&c.File, "file", "f", c.File, "Load nodegroup data from file")
	fs.BoolVarP(&c.DoNotDelete, "do-not-delete", "", c.DoNotDelete, "Set do not delete flag")
	fs.StringVarP(&c.Output, "output", "o", c.Output, "Output format. One of: yaml|json.")
}

func (c *NodeGroupEditConfig) ValidateFlags(cmd *cobra.Command, args []string) error {
	flags.EnsureRequiredFlags(cmd, "cluster")
	if len(args) == 0 {
		return errors.New("missing nodegroup name")
	}
	if len(args) > 1 {
		return errors.New("multiple nodegroup name provided")
	}
	c.ClusterName = strings.ToLower(c.ClusterName)
	c.NgName = args[0]
	return nil
}
