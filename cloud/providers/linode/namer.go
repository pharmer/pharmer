package linode

import (
	"strconv"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/pharmer/api"
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

func (n namer) StartupScriptName(sku, role string) string {
	return n.cluster.Name + "-" + sku + "-" + role + "-V" + strconv.FormatInt(n.cluster.Generation, 10)
}
