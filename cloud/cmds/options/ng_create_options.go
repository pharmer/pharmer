package options

import (
	"fmt"
	"strings"

	"github.com/appscode/go/flags"
	api "github.com/pharmer/pharmer/apis/v1alpha1"
	"github.com/pharmer/pharmer/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type NodeGroupCreateConfig struct {
	ClusterName  string
	NodeType     string
	SpotPriceMax float64
	Nodes        map[string]int
	Owner        string
}

func NewNodeGroupCreateConfig() *NodeGroupCreateConfig {
	return &NodeGroupCreateConfig{
		ClusterName:  "",
		NodeType:     string(api.NodeTypeRegular),
		SpotPriceMax: float64(0),
		Nodes:        map[string]int{},
		Owner:        utils.GetLocalOwner(),
	}
}

func (c *NodeGroupCreateConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&c.ClusterName, "cluster", "k", c.ClusterName, "Name of the Kubernetes cluster")
	fs.StringVar(&c.NodeType, "type", c.NodeType, "Set node type regular/spot, default regular")
	fs.Float64Var(&c.SpotPriceMax, "spot-price-max", c.SpotPriceMax, "Maximum price of spot instance")
	fs.StringToIntVar(&c.Nodes, "nodes", c.Nodes, "Node set configuration")

	fs.StringVarP(&c.Owner, "owner", "o", c.Owner, "Current user id")
}

func (c *NodeGroupCreateConfig) ValidateFlags(cmd *cobra.Command, args []string) error {
	ensureFlags := []string{"cluster", "nodes"}
	if api.NodeType(c.NodeType) == api.NodeTypeSpot {
		ensureFlags = append(ensureFlags, "spot-price-max")
	}
	flags.EnsureRequiredFlags(cmd, ensureFlags...)

	c.ClusterName = strings.ToLower(c.ClusterName)
	switch api.NodeType(c.NodeType) {
	case api.NodeTypeSpot, api.NodeTypeRegular:
		break
	default:
		return errors.New(fmt.Sprintf("flag [type] must be %v or %v", api.NodeTypeRegular, api.NodeTypeSpot))
	}
	return nil
}
