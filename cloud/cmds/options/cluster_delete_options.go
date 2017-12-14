package options

import (
	"errors"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type ClusterDeleteConfig struct {
	Force                bool
	ReleaseReservedIP    bool
	KeepLBs              bool
	DeleteDynamicVolumes bool
}

func NewClusterDeleteConfig() *ClusterDeleteConfig {
	return &ClusterDeleteConfig{
		ReleaseReservedIP:    false,
		Force:                false,
		KeepLBs:              false,
		DeleteDynamicVolumes: false,
	}
}

func (c *ClusterDeleteConfig) AddClusterDeleteFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&c.Force, "force", c.Force, "Force delete any running non-system apps")
	fs.BoolVar(&c.ReleaseReservedIP, "release-reserved-ip", c.ReleaseReservedIP, "Release reserved IP")
	fs.BoolVar(&c.KeepLBs, "keep-loadbalancers", c.KeepLBs, "Keep loadbalancers")
	fs.BoolVar(&c.DeleteDynamicVolumes, "delete-dynamic-volumes", c.DeleteDynamicVolumes, "Delete dynamically provisioned volumes")

}

func (c *ClusterDeleteConfig) ValidateClusterDeleteFlags(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		errors.New("missing cluster name")
	}
	return nil
}
