package scaleway

import (
	"github.com/appscode/go/crypto/rand"
	api "github.com/appscode/pharmer/apis/v1alpha1"
)

type namer struct {
	cluster *api.Cluster
}

func (n namer) MasterName() string {
	return n.cluster.Name + "-master"
}

func (n namer) GenNodeName() string {
	return rand.WithUniqSuffix(n.cluster.Name + "-node")
}

func (n namer) GenSSHKeyExternalID() string {
	return n.cluster.Name + "-" + rand.Characters(6)
}
