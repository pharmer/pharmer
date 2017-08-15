package vultr

import (
	"strconv"

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

func (n namer) ReserveIPName() string {
	return n.ctx.Name + "-master-ip"
}

func (n namer) StartupScriptName(sku, role string) string {
	return n.ctx.Name + "-" + sku + "-" + role + "-V" + strconv.FormatInt(n.ctx.ContextVersion, 10)
}
