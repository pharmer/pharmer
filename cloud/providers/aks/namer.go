package aks

import (
	"strings"

	"github.com/appscode/go/crypto/rand"
	api "github.com/pharmer/pharmer/apis/v1beta1"
)

type namer struct {
	cluster *api.Cluster
}

func (n namer) GenSSHKeyExternalID() string {
	return n.cluster.Name + "-" + rand.Characters(6)
}

func (n namer) ResourceGroupName() string {
	return n.cluster.Name
}

func (n namer) AvailabilitySetName() string {
	return n.cluster.Name + "-as"
}

func (n namer) VirtualNetworkName() string {
	return n.cluster.Name + "-vnet"
}

func (n namer) NetworkSecurityGroupName() string {
	return n.cluster.Name + "-nsg"
}

func (n namer) SubnetName() string {
	return n.cluster.Name + "-subnet"
}

func (n namer) RouteTableName() string {
	return n.cluster.Name + "-rt"
}

func (n namer) GenStorageAccountName() string {
	return strings.Replace("k8s-"+rand.WithUniqSuffix(n.cluster.Name), "-", "", -1)
}

func (n namer) AdminUsername() string {
	return "kube"
}

func (n namer) GetNodeGroupName(ng string) string {
	name := strings.ToLower(ng)
	name = strings.Replace(name, "standard", "s", -1)
	name = strings.Replace(name, "pool", "p", -1)
	name = strings.Replace(name, "-", "", -1)
	return name
}
