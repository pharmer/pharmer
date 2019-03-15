package dokube

import (
	"github.com/appscode/go/crypto/rand"
	api "github.com/pharmer/pharmer/apis/v1alpha1"
)

type namer struct {
	cluster *api.Cluster
}

func (n namer) AdminUsername() string {
	return "pharmer"
}

func (n namer) GenSSHKeyExternalID() string {
	return n.cluster.Name + "-" + rand.Characters(6)
}
