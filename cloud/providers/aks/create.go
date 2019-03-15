package aks

import (
	"encoding/json"
	"net"

	containersvc "github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2017-09-30/containerservice"
	"github.com/appscode/go/crypto/rand"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) GetDefaultMachineProviderSpec(cluster *api.Cluster, sku string, role api.MachineRole) (clusterapi.ProviderSpec, error) {
	if sku == "" {
		sku = "Standard_D2_v2"
	}
	spec := &api.AKSMachineProviderSpec{
		Roles:    []api.MachineRole{role},
		Location: cluster.Spec.Config.Cloud.Zone,
		OSDisk: api.OSDisk{

			OSType:     string(containersvc.Linux),
			DiskSizeGB: 30,
			ManagedDisk: api.ManagedDisk{
				StorageAccountType: "Premium_LRS",
			},
		},
		VMSize: sku,
		Image: api.Image{
			Publisher: "Canonical",
			Offer:     "UbuntuServer",
			SKU:       "16.04-LTS",
			Version:   "latest",
		},
		SSHPublicKey:  string(SSHKey(cm.ctx).PublicKey),
		SSHPrivateKey: string(SSHKey(cm.ctx).PrivateKey),
	}
	providerSpecValue, err := json.Marshal(spec)
	if err != nil {
		return clusterapi.ProviderSpec{}, err
	}

	return clusterapi.ProviderSpec{
		Value: &runtime.RawExtension{
			Raw: providerSpecValue,
		},
	}, nil
}

func (cm *ClusterManager) SetOwner(owner string) {
	cm.owner = owner
}

func (cm *ClusterManager) SetDefaultCluster(cluster *api.Cluster, config *api.ClusterConfig) error {
	n := namer{cluster: cluster}

	if err := api.AssignTypeKind(cluster); err != nil {
		return err
	}
	if err := api.AssignTypeKind(cluster.Spec.ClusterAPI); err != nil {
		return err
	}

	// Init spec
	config.Cloud.Region = config.Cloud.Zone
	config.Cloud.SSHKeyName = n.GenSSHKeyExternalID()

	cluster.SetNetworkingDefaults(config.Cloud.NetworkProvider)

	config.Cloud.Azure = &api.AzureSpec{
		ResourceGroup:      n.ResourceGroupName(),
		SubnetName:         n.SubnetName(),
		SecurityGroupName:  n.NetworkSecurityGroupName(),
		VnetName:           n.VirtualNetworkName(),
		RouteTableName:     n.RouteTableName(),
		StorageAccountName: n.GenStorageAccountName(),
		SubnetCIDR:         "10.240.0.0/16",
		RootPassword:       rand.GeneratePassword(),
	}

	// Init status
	cluster.Status = api.PharmerClusterStatus{
		Phase: api.ClusterPending,
	}

	return cluster.SetAKSProviderConfig(cluster.Spec.ClusterAPI, config)

}

func (cm *ClusterManager) IsValid(cluster *api.Cluster) (bool, error) {
	return false, ErrNotImplemented
}

func (cm *ClusterManager) GetSSHConfig(cluster *api.Cluster, node *core.Node) (*api.SSHConfig, error) {
	cfg := &api.SSHConfig{
		PrivateKey: SSHKey(cm.ctx).PrivateKey,
		User:       "ubuntu",
		HostPort:   int32(22),
	}
	for _, addr := range node.Status.Addresses {
		if addr.Type == core.NodeExternalIP {
			cfg.HostIP = addr.Address
		}
	}
	if net.ParseIP(cfg.HostIP) == nil {
		return nil, errors.Errorf("failed to detect external Ip for node %s of cluster %s", node.Name, cluster.Name)
	}
	return cfg, nil
}
