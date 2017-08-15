package digitalocean

import (
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/pharmer/api"
)

type namer struct {
	ctx *api.Cluster
}

func (n namer) MasterName() string {
	return n.ctx.Name + "-master"
}

func (n namer) GenNodeName(sku string) string {
	return rand.WithUniqSuffix(n.GetInstanceGroupName(sku))
}

func (n namer) GenSSHKeyExternalID() string {
	return n.ctx.Name + "-" + rand.Characters(6)
}

func (n namer) GetInstanceGroupName(sku string) string {
	return n.ctx.Name + "-" + sku + "-node"

}
