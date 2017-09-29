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

// Deprecated
func (n namer) GenNodeName(ng string) string {
	return rand.WithUniqSuffix(ng)
}

func (n namer) GenSSHKeyExternalID() string {
	return n.cluster.Name + "-" + rand.Characters(6)
}
