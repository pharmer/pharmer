package azure

import (
	"fmt"
	"hash/fnv"
	"strings"

	"github.com/appscode/go/crypto/rand"
	api "github.com/pharmer/pharmer/apis/v1beta1"
)

type namer struct {
	cluster *api.Cluster
}

const (
	// DefaultUserName is the default username for created vm
	DefaultUserName = "capi"
	// DefaultVnetCIDR is the default Vnet CIDR
	DefaultVnetCIDR = "10.0.0.0/8"
	// DefaultControlPlaneSubnetCIDR is the default Control Plane Subnet CIDR
	DefaultControlPlaneSubnetCIDR = "10.0.0.0/16"
	// DefaultNodeSubnetCIDR is the default Node Subnet CIDR
	DefaultNodeSubnetCIDR = "10.1.0.0/16"
	// DefaultInternalLBIPAddress is the default internal load balancer ip address
	DefaultInternalLBIPAddress = "10.0.0.100"
	// DefaultAzureDNSZone is the default provided azure dns zone
	DefaultAzureDNSZone = "cloudapp.azure.com"
)

//ref: https://github.com/kubernetes-sigs/cluster-api-provider-azure/blob/c4896544d32792f06f5302c4fd9d2b4fdff358e1/pkg/cloud/azure/defaults.go#L35-L34

// GenerateVnetName generates a virtual network name, based on the cluster name.
func (n namer) GenerateVnetName() string {
	return fmt.Sprintf("%s-%s", n.cluster.Name, "vnet")
}

// GenerateControlPlaneSecurityGroupName generates a control plane security group name, based on the cluster name.
func (n namer) GenerateControlPlaneSecurityGroupName() string {
	return fmt.Sprintf("%s-%s", n.cluster.Name, "controlplane-nsg")
}

// GenerateNodeSecurityGroupName generates a node security group name, based on the cluster name.
func (n namer) GenerateNodeSecurityGroupName() string {
	return fmt.Sprintf("%s-%s", n.cluster.Name, "node-nsg")
}

// GenerateNodeRouteTableName generates a node route table name, based on the cluster name.
func (n namer) GenerateNodeRouteTableName() string {
	return fmt.Sprintf("%s-%s", n.cluster.Name, "node-routetable")
}

// GenerateControlPlaneSubnetName generates a node subnet name, based on the cluster name.
func (n namer) GenerateControlPlaneSubnetName() string {
	return fmt.Sprintf("%s-%s", n.cluster.Name, "controlplane-subnet")
}

// GenerateNodeSubnetName generates a node subnet name, based on the cluster name.
func (n namer) GenerateNodeSubnetName() string {
	return fmt.Sprintf("%s-%s", n.cluster.Name, "node-subnet")
}

// GenerateInternalLBName generates a internal load balancer name, based on the cluster name.
func (n namer) GenerateInternalLBName() string {
	return fmt.Sprintf("%s-%s", n.cluster.Name, "internal-lb")
}

// GeneratePublicLBName generates a public load balancer name, based on the cluster name.
func (n namer) GeneratePublicLBName() string {
	return fmt.Sprintf("%s-%s", n.cluster.Name, "public-lb")
}

// GeneratePublicIPName generates a public IP name, based on the cluster name and a hash.
func (n namer) GeneratePublicIPName() string {
	h := fnv.New32a()
	h.Write([]byte(fmt.Sprintf("%s/%s/%s", n.cluster.Spec.Config.Cloud.Azure.SubscriptionID, n.ResourceGroupName(), n.cluster.Name)))
	hash := fmt.Sprintf("%x", h.Sum32())

	return fmt.Sprintf("%s-%s", n.cluster.Name, hash)
}

// GenerateFQDN generates a fully qualified domain name, based on the public IP name and cluster location.
func (n namer) GenerateFQDN(publicIPName, location string) string {
	return fmt.Sprintf("%s.%s.%s", publicIPName, location, DefaultAzureDNSZone)
}

func (n namer) MasterName() string {
	return n.cluster.Name + "-master"
}

func (n namer) GenSSHKeyExternalID() string {
	return n.cluster.Name + "-sshkey"
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
	return fmt.Sprintf("%s-%s", n.cluster.Name, "vnet")
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
	return DefaultUserName
}

func (n namer) BlobName(instanceName string) string {
	return instanceName + "-osdisk.vhd"

}
