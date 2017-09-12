package digitalocean

import (
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/pharmer/api"
)

type namer struct {
	cluster *api.Cluster
}

func (n namer) MasterName() string {
	return n.cluster.Name + "-master"
}

func (n namer) GenNodeName(sku string) string {
	return rand.WithUniqSuffix(n.GetNodeGroupName(sku))
}

func (n namer) GenSSHKeyExternalID() string {
	return n.cluster.Name + "-" + rand.Characters(6)
}

func (n namer) GetNodeGroupName(sku string) string {
	return n.cluster.Name + "-" + sku + "-node"

}
