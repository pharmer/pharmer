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
	return rand.WithUniqSuffix(n.GetNodeSetName(sku))
}

func (n namer) GenSSHKeyExternalID() string {
	return n.cluster.Name + "-" + rand.Characters(6)
}

func (n namer) GetNodeSetName(sku string) string {
	return n.cluster.Name + "-" + sku + "-node"

}
