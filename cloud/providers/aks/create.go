package aks

import (
	"encoding/json"

	containersvc "github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2019-06-01/containerservice"
	"github.com/appscode/go/crypto/rand"
	"k8s.io/apimachinery/pkg/runtime"
	api "pharmer.dev/pharmer/apis/v1beta1"
	"pharmer.dev/pharmer/apis/v1beta1/azure"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) GetDefaultMachineProviderSpec(sku string, role api.MachineRole) (clusterapi.ProviderSpec, error) {
	cluster := cm.Cluster
	certs := cm.Certs
	if sku == "" {
		sku = "Standard_D2_v2"
	}
	spec := &azure.AzureMachineProviderSpec{
		Roles:    []azure.MachineRole{azure.MachineRole(role)},
		Location: cluster.Spec.Config.Cloud.Zone,
		OSDisk: azure.OSDisk{

			OSType:     string(containersvc.Linux),
			DiskSizeGB: 30,
			ManagedDisk: azure.ManagedDisk{
				StorageAccountType: "Premium_LRS",
			},
		},
		VMSize: sku,
		Image: azure.Image{
			Publisher: "Canonical",
			Offer:     "UbuntuServer",
			SKU:       "16.04-LTS",
			Version:   "latest",
		},
		SSHPublicKey:  string(certs.SSHKey.PublicKey),
		SSHPrivateKey: string(certs.SSHKey.PrivateKey),
	}
	providerSpecValue, err := json.Marshal(spec)
	if err != nil {
		cm.Logger.Error(err, "failed to marshal provider spec")
		return clusterapi.ProviderSpec{}, err
	}

	return clusterapi.ProviderSpec{
		Value: &runtime.RawExtension{
			Raw: providerSpecValue,
		},
	}, nil
}

func (cm *ClusterManager) SetDefaultCluster() error {
	cluster := cm.Cluster
	n := namer{cluster: cluster}

	cluster.Spec.Config.Cloud.Azure = &api.AzureSpec{
		ResourceGroup:      n.ResourceGroupName(),
		SubnetName:         n.SubnetName(),
		SecurityGroupName:  n.NetworkSecurityGroupName(),
		VnetName:           n.VirtualNetworkName(),
		RouteTableName:     n.RouteTableName(),
		StorageAccountName: n.GenStorageAccountName(),
		SubnetCIDR:         "10.240.0.0/16",
		RootPassword:       rand.GeneratePassword(),
	}
	cluster.Spec.Config.SSHUserName = "ubuntu"

	return nil
}
