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

type ClusterDeleteConfig struct {
	Force                bool
	ReleaseReservedIP    bool
	KeepLBs              bool
	DeleteDynamicVolumes bool
	Clusters             []string
}

func NewClusterDeleteConfig() *ClusterDeleteConfig {
	return &ClusterDeleteConfig{
		ReleaseReservedIP:    false,
		Force:                false,
		KeepLBs:              false,
		DeleteDynamicVolumes: false,
	}
}

func (c *ClusterDeleteConfig) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&c.Force, "force", c.Force, "Force delete any running non-system apps")
	fs.BoolVar(&c.ReleaseReservedIP, "release-reserved-ip", c.ReleaseReservedIP, "Release reserved IP")
	fs.BoolVar(&c.KeepLBs, "keep-loadbalancers", c.KeepLBs, "Keep loadbalancers")
	fs.BoolVar(&c.DeleteDynamicVolumes, "delete-dynamic-volumes", c.DeleteDynamicVolumes, "Delete dynamically provisioned volumes")

}

func (c *ClusterDeleteConfig) ValidateFlags(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("missing cluster name")
	}
	c.Clusters = func(names []string) []string {
		for i := range names {
			names[i] = strings.ToLower(names[i])
		}
		return names
	}(args)
	return nil
}
