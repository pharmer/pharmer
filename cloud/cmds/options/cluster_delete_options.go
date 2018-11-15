package options

import (
	"strings"

	"github.com/pharmer/pharmer/utils"
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
	Owner                string
}

func NewClusterDeleteConfig() *ClusterDeleteConfig {
	return &ClusterDeleteConfig{
		ReleaseReservedIP:    false,
		Force:                false,
		KeepLBs:              false,
		DeleteDynamicVolumes: false,
		Owner:                utils.GetLocalOwner(),
	}
}

func (c *ClusterDeleteConfig) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&c.Force, "force", c.Force, "Force delete any running non-system apps")
	fs.BoolVar(&c.ReleaseReservedIP, "release-reserved-ip", c.ReleaseReservedIP, "Release reserved IP")
	fs.BoolVar(&c.KeepLBs, "keep-loadbalancers", c.KeepLBs, "Keep loadbalancers")
	fs.BoolVar(&c.DeleteDynamicVolumes, "delete-dynamic-volumes", c.DeleteDynamicVolumes, "Delete dynamically provisioned volumes")
	fs.StringVarP(&c.Owner, "owner", "o", c.Owner, "Current user id")

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
