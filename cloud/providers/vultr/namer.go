package vultr

import (
	"strconv"

	"github.com/appscode/go/crypto/rand"
	api "github.com/appscode/pharmer/apis/v1alpha1"
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

func (n namer) ReserveIPName() string {
	return n.cluster.Name + "-master-ip"
}

func (n namer) StartupScriptName(sku, role string) string {
	return n.cluster.Name + "-" + sku + "-" + role + "-V" + strconv.FormatInt(n.cluster.Generation, 10)
}

func (n namer) GetNodeGroupName(sku string) string {
	return n.cluster.Name + "-" + sku
}
