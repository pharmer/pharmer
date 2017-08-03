package azure

import (
	"strings"

	"github.com/Azure/azure-sdk-for-go/arm/storage"
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/types"
	"github.com/appscode/pharmer/contexts"
)

type namer struct {
	ctx *contexts.ClusterContext
}

func (n namer) MasterName() string {
	return n.ctx.Name + "-master"
}

func (n namer) GenSSHKeyExternalID() string {
	return n.ctx.Name + "-" + rand.Characters(6)
}

func (n namer) GenNodeName(sku string) string {
	return rand.WithUniqSuffix(n.ctx.Name + "-" + strings.Replace(sku, "_", "-", -1) + "-node")
}

func (n namer) NetworkInterfaceName(instanceName string) string {
	return instanceName + "-nic"
}

func (n namer) PublicIPName(instanceName string) string {
	return instanceName + "-pip"
}

func (n namer) ResourceGroupName() string {
	return n.ctx.Name
}

func (n namer) AvailablitySetName() string {
	return n.ctx.Name + "-as"
}

func (n namer) VirtualNetworkName() string {
	return n.ctx.Name + "-vnet"
}

func (n namer) NetworkSecurityGroupName() string {
	return n.ctx.Name + "-nsg"
}

func (n namer) SubnetName() string {
	return n.ctx.Name + "-subnet"
}

func (n namer) RouteTableName() string {
	return n.ctx.Name + "-rt"
}

func (n namer) NetworkSecurityRule(protocol string) string {
	return n.ctx.Name + "-" + protocol
}

func (n namer) GenStorageAccountName() string {
	return strings.Replace("k8s-"+rand.WithUniqSuffix(n.ctx.Name), "-", "", -1)
}

func (n namer) StorageContainerName() string {
	return n.ctx.Name + "-data"
}

func (n namer) GetInstanceGroupName(sku string) string {
	return n.ctx.Name + "-" + strings.Replace(sku, "_", "-", -1) + "-node"

}

func (n namer) AdminUsername() string {
	return "kube"
}

func (n namer) BootDiskName(instanceName string) string {
	return instanceName + "-osdisk"
}

// https://k1g09f7j8mf0htzaq7mq4k8s.blob.core.windows.net/strgkubernetes/kubernetes-master-osdisk.vhd
func (n namer) BootDiskURI(sa storage.Account, instanceName string) string {
	return types.String(sa.PrimaryEndpoints.Blob) + n.ctx.AzureCloudConfig.StorageAccountName + "/" + instanceName + "-osdisk.vhd"
}

func (n namer) BlobName(instanceName string) string {
	return instanceName + "-osdisk.vhd"

}
