package azure

import (
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2017-10-01/storage"
	"github.com/appscode/go/crypto/rand"
	. "github.com/appscode/go/types"
	api "github.com/pharmer/pharmer/apis/v1beta1"
)

type namer struct {
	cluster *api.Cluster
}

func (n namer) MasterName() string {
	return n.cluster.Name + "-master"
}

func (n namer) GenSSHKeyExternalID() string {
	return n.cluster.Name + "-" + rand.Characters(6)
}

func (n namer) GenNodeName(ng string) string {
	return rand.WithUniqSuffix(ng)
}

func (n namer) NetworkInterfaceName(instanceName string) string {
	return instanceName + "-nic"
}

func (n namer) PublicIPName(instanceName string) string {
	return instanceName + "-api"
}

func (n namer) ResourceGroupName() string {
	return n.cluster.Name
}

func (n namer) AvailabilitySetName() string {
	return n.cluster.Name + "-as"
}

func (n namer) VirtualNetworkName() string {
	return "ClusterAPIVnet"
}

func (n namer) NetworkSecurityGroupName() string {
	return "ClusterAPINSG"
}

func (n namer) SubnetName() string {
	return "ClusterAPISubnet"
}

func (n namer) LoadBalancerName() string {
	return n.cluster.Name + "-api"
}

func (n namer) RouteTableName() string {
	return n.cluster.Name + "-rt"
}

func (n namer) NetworkSecurityRule(protocol string) string {
	return n.cluster.Name + "-" + protocol
}

func (n namer) GenStorageAccountName() string {
	return strings.Replace("k8s-"+rand.WithUniqSuffix(n.cluster.Name), "-", "", -1)
}

func (n namer) StorageContainerName() string {
	return n.cluster.Name + "-data"
}

func (n namer) AdminUsername() string {
	return "kube"
}

func (n namer) BootDiskName(instanceName string) string {
	return instanceName + "-osdisk"
}

// https://k1g09f7j8mf0htzaq7mq4k8s.blob.core.windows.net/strgkubernetes/kubernetes-master-osdisk.vhd
func (n namer) BootDiskURI(sa storage.Account, instanceName string) string {
	return String(sa.PrimaryEndpoints.Blob) + n.cluster.Spec.Config.Cloud.Azure.StorageAccountName + "/" + instanceName + "-osdisk.vhd"
}

func (n namer) BlobName(instanceName string) string {
	return instanceName + "-osdisk.vhd"

}
