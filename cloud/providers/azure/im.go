package azure

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/Azure/azure-sdk-for-go/arm/network"
	armstorage "github.com/Azure/azure-sdk-for-go/arm/storage"
	azstore "github.com/Azure/azure-sdk-for-go/storage"
	_env "github.com/appscode/go/env"
	"github.com/appscode/go/errors"
	. "github.com/appscode/go/types"
	"github.com/appscode/pharmer/api"
	. "github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/credential"
	"github.com/appscode/pharmer/phid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	machineIDTemplate = "/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s"
)

type instanceManager struct {
	ctx     context.Context
	cluster *api.Cluster
	conn    *cloudConnector
	namer   namer
}

func (im *instanceManager) GetInstance(md *api.NodeStatus) (*api.Node, error) {
	pip, err := im.conn.publicIPAddressesClient.Get(im.namer.ResourceGroupName(), im.namer.PublicIPName(md.Name), "")
	if err != nil {
		return nil, errors.FromErr(err).WithContext(im.ctx).Err()
	}

	nic, err := im.conn.interfacesClient.Get(im.namer.ResourceGroupName(), im.namer.NetworkInterfaceName(md.Name), "")
	if err != nil {
		return nil, errors.FromErr(err).WithContext(im.ctx).Err()
	}

	vm, err := im.conn.vmClient.Get(im.namer.ResourceGroupName(), md.Name, compute.InstanceView)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(im.ctx).Err()
	}

	i, err := im.newKubeInstance(vm, nic, pip)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(im.ctx).Err()
	}
	// TODO: Role not set
	return i, nil
}

func (im *instanceManager) createPublicIP(name string, alloc network.IPAllocationMethod) (network.PublicIPAddress, error) {
	req := network.PublicIPAddress{
		Name:     StringP(name),
		Location: StringP(im.cluster.Spec.Cloud.Zone),
		PublicIPAddressPropertiesFormat: &network.PublicIPAddressPropertiesFormat{
			PublicIPAllocationMethod: alloc,
		},
		Tags: &map[string]*string{
			"KubernetesCluster": StringP(im.cluster.Name),
		},
	}

	_, errchan := im.conn.publicIPAddressesClient.CreateOrUpdate(im.namer.ResourceGroupName(), name, req, nil)
	err := <-errchan
	if err != nil {
		return network.PublicIPAddress{}, err
	}
	Logger(im.ctx).Infof("Public ip addres %v created", name)
	return im.conn.publicIPAddressesClient.Get(im.namer.ResourceGroupName(), name, "")
}

func (im *instanceManager) getPublicIP(name string) (network.PublicIPAddress, error) {
	return im.conn.publicIPAddressesClient.Get(im.namer.ResourceGroupName(), name, "")
}

func (im *instanceManager) getAvailablitySet() (compute.AvailabilitySet, error) {
	setName := im.namer.AvailablitySetName()
	return im.conn.availabilitySetsClient.Get(im.namer.ResourceGroupName(), setName)
}

func (im *instanceManager) getStorageAccount() (armstorage.Account, error) {
	storageName := im.cluster.Spec.Cloud.Azure.CloudConfig.StorageAccountName
	account, err := im.conn.storageClient.GetProperties(im.namer.ResourceGroupName(), storageName)
	return account, err
}

func (im *instanceManager) createNetworkInterface(name string, sg network.SecurityGroup, subnet network.Subnet, alloc network.IPAllocationMethod, internalIP string, pip network.PublicIPAddress) (network.Interface, error) {
	req := network.Interface{
		Name:     StringP(name),
		Location: StringP(im.cluster.Spec.Cloud.Zone),
		InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
			IPConfigurations: &[]network.InterfaceIPConfiguration{
				{
					Name: StringP("ipconfig"),
					InterfaceIPConfigurationPropertiesFormat: &network.InterfaceIPConfigurationPropertiesFormat{
						Subnet: &network.Subnet{
							ID: subnet.ID,
						},
						PrivateIPAllocationMethod: alloc,
						PublicIPAddress: &network.PublicIPAddress{
							ID: pip.ID,
						},
					},
				},
			},
			EnableIPForwarding: TrueP(),
			NetworkSecurityGroup: &network.SecurityGroup{
				ID: sg.ID,
			},
		},
		Tags: &map[string]*string{
			"KubernetesCluster": StringP(im.cluster.Name),
		},
	}
	if alloc == network.Static {
		if internalIP == "" {
			return network.Interface{}, errors.New("No private IP provided for Static allocation.").WithContext(im.ctx).Err()
		}
		(*req.IPConfigurations)[0].PrivateIPAddress = StringP(internalIP)
	}
	_, errchan := im.conn.interfacesClient.CreateOrUpdate(im.namer.ResourceGroupName(), name, req, nil)
	err := <-errchan
	if err != nil {
		return network.Interface{}, err
	}
	Logger(im.ctx).Infof("Network interface %v created", name)
	return im.conn.interfacesClient.Get(im.namer.ResourceGroupName(), name, "")
}

func (im *instanceManager) createVirtualMachine(nic network.Interface, as compute.AvailabilitySet, sa armstorage.Account, vmName, data, vmSize string) (compute.VirtualMachine, error) {
	req := compute.VirtualMachine{
		Name:     StringP(vmName),
		Location: StringP(im.cluster.Spec.Cloud.Zone),
		VirtualMachineProperties: &compute.VirtualMachineProperties{
			AvailabilitySet: &compute.SubResource{
				ID: as.ID,
			},
			NetworkProfile: &compute.NetworkProfile{
				NetworkInterfaces: &[]compute.NetworkInterfaceReference{
					{
						ID: nic.ID,
					},
				},
			},
			OsProfile: &compute.OSProfile{
				ComputerName:  StringP(vmName),
				AdminPassword: StringP(im.cluster.Spec.Cloud.Azure.InstanceRootPassword),
				AdminUsername: StringP(im.namer.AdminUsername()),
				CustomData:    StringP(base64.StdEncoding.EncodeToString([]byte(data))),
				LinuxConfiguration: &compute.LinuxConfiguration{
					DisablePasswordAuthentication: BoolP(!_env.FromHost().DebugEnabled()),
					SSH: &compute.SSHConfiguration{
						PublicKeys: &[]compute.SSHPublicKey{
							{
								KeyData: StringP(string(SSHKey(im.ctx).PublicKey)),
								Path:    StringP(fmt.Sprintf("/home/%v/.ssh/authorized_keys", im.namer.AdminUsername())),
							},
						},
					},
				},
			},
			StorageProfile: &compute.StorageProfile{
				ImageReference: &compute.ImageReference{
					Publisher: StringP(im.cluster.Spec.Cloud.InstanceImageProject),
					Offer:     StringP(im.cluster.Spec.Cloud.OS),
					Sku:       StringP(im.cluster.Spec.Cloud.InstanceImage),
					Version:   StringP(im.cluster.Spec.Cloud.Azure.InstanceImageVersion),
				},
				OsDisk: &compute.OSDisk{
					Caching:      compute.ReadWrite,
					CreateOption: compute.FromImage,
					Name:         StringP(im.namer.BootDiskName(vmName)),
					Vhd: &compute.VirtualHardDisk{
						URI: StringP(im.namer.BootDiskURI(sa, vmName)),
					},
				},
			},
			HardwareProfile: &compute.HardwareProfile{
				VMSize: compute.VirtualMachineSizeTypes(vmSize),
			},
		},
		Tags: &map[string]*string{
			"KubernetesCluster": StringP(im.cluster.Name),
		},
	}

	_, errchan := im.conn.vmClient.CreateOrUpdate(im.namer.ResourceGroupName(), vmName, req, nil)
	err := <-errchan
	if err != nil {
		return compute.VirtualMachine{}, err
	}
	Logger(im.ctx).Infof("Virtual machine with disk %v password %v created", im.namer.BootDiskURI(sa, vmName), im.cluster.Spec.Cloud.Azure.InstanceRootPassword)
	// https://docs.microsoft.com/en-us/azure/virtual-machines/virtual-machines-linux-extensions-customscript?toc=%2fazure%2fvirtual-machines%2flinux%2ftoc.json
	// https://github.com/Azure/custom-script-extension-linux
	// old: https://github.com/Azure/azure-linux-extensions/tree/master/CustomScript
	// https://docs.microsoft.com/en-us/azure/virtual-machines/virtual-machines-windows-classic-inject-custom-data
	Logger(im.ctx).Infof("Running startup script in virtual machine %v", vmName)
	extName := vmName + "-script"
	extReq := compute.VirtualMachineExtension{
		Name:     StringP(extName),
		Type:     StringP("Microsoft.Compute/virtualMachines/extensions"),
		Location: StringP(im.cluster.Spec.Cloud.Zone),
		VirtualMachineExtensionProperties: &compute.VirtualMachineExtensionProperties{
			Publisher:               StringP("Microsoft.Azure.Extensions"),
			Type:                    StringP("CustomScript"),
			TypeHandlerVersion:      StringP("2.0"),
			AutoUpgradeMinorVersion: TrueP(),
			Settings: &map[string]interface{}{
				"commandToExecute": "cat /var/lib/waagent/CustomData | base64 --decode | /bin/bash",
			},
			// ProvisioningState
		},
		Tags: &map[string]*string{
			"KubernetesCluster": StringP(im.cluster.Name),
		},
	}
	_, errchan = im.conn.vmExtensionsClient.CreateOrUpdate(im.namer.ResourceGroupName(), vmName, extName, extReq, nil)
	err = <-errchan
	if err != nil {
		return compute.VirtualMachine{}, err
	}

	//Logger(im.ctx).Infof("Restarting virtual machine %v", vmName)
	//_, err = im.conn.vmClient.Restart(im.namer.ResourceGroupName(), vmName, nil)
	//if err != nil {
	//	return compute.VirtualMachine{}, err
	//}

	vm, err := im.conn.vmClient.Get(im.namer.ResourceGroupName(), vmName, compute.InstanceView)
	Logger(im.ctx).Infof("Found virtual machine %v", vm)
	return vm, err
}

func (im *instanceManager) DeleteVirtualMachine(vmName string) error {
	_, errchan := im.conn.vmClient.Delete(im.namer.ResourceGroupName(), vmName, nil)
	err := <-errchan
	if err != nil {
		return err
	}
	storageName := im.cluster.Spec.Cloud.Azure.CloudConfig.StorageAccountName
	keys, err := im.conn.storageClient.ListKeys(im.namer.ResourceGroupName(), storageName)
	if err != nil {
		return err
	}
	Logger(im.ctx).Infof("Virtual machine %v deleted", vmName)
	storageClient, err := azstore.NewBasicClient(storageName, *(*(keys.Keys))[0].Value)
	if err != nil {
		return err
	}
	bs := storageClient.GetBlobService()
	_, err = bs.GetContainerReference(storageName).GetBlobReference(im.namer.BlobName(vmName)).DeleteIfExists(nil)
	return err
}

func (im *instanceManager) newKubeInstance(vm compute.VirtualMachine, nic network.Interface, pip network.PublicIPAddress) (*api.Node, error) {
	// TODO: Load once
	cred, err := Store(im.ctx).Credentials().Get(im.cluster.Spec.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.Azure{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.New().WithMessagef("Credential %s is invalid. Reason: %v", im.cluster.Spec.CredentialName, err)
	}

	i := api.Node{
		ObjectMeta: metav1.ObjectMeta{
			UID:  phid.NewKubeInstance(),
			Name: *vm.Name,
		},
		Spec: api.NodeSpec{
			SKU: string(vm.HardwareProfile.VMSize),
		},
		Status: api.NodeStatus{
			ExternalID:    fmt.Sprintf(machineIDTemplate, typed.SubscriptionID(), im.namer.ResourceGroupName(), *vm.Name),
			ExternalPhase: *vm.ProvisioningState,
			PrivateIP:     *(*nic.IPConfigurations)[0].PrivateIPAddress,
			Phase:         api.NodeReady,
		},
	}
	if pip.IPAddress != nil {
		i.Status.PublicIP = *pip.IPAddress
	}
	return &i, nil
}
