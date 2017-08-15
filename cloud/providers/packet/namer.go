package packet

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

func (n namer) GenNodeName() string {
	return rand.WithUniqSuffix(n.ctx.Name + "-node")
}

func (n namer) GenSSHKeyExternalID() string {
	return n.ctx.Name + "-" + rand.Characters(6)
}
