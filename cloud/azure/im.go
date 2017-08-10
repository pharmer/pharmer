package azure

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/Azure/azure-sdk-for-go/arm/network"
	armstorage "github.com/Azure/azure-sdk-for-go/arm/storage"
	azstore "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/appscode/errors"
	_env "github.com/appscode/go/env"
	"github.com/appscode/go/types"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud/lib"
	"github.com/appscode/pharmer/contexts"
	"github.com/appscode/pharmer/phid"
	"github.com/appscode/pharmer/storage"
	"github.com/appscode/pharmer/util/credentialutil"
)

const (
	machineIDTemplate = "/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s"
)

type instanceManager struct {
	ctx   *contexts.ClusterContext
	conn  *cloudConnector
	namer namer
}

func (im *instanceManager) GetInstance(md *contexts.InstanceMetadata) (*contexts.KubernetesInstance, error) {
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
		Name:     types.StringP(name),
		Location: types.StringP(im.ctx.Zone),
		PublicIPAddressPropertiesFormat: &network.PublicIPAddressPropertiesFormat{
			PublicIPAllocationMethod: alloc,
		},
		Tags: &map[string]*string{
			"KubernetesCluster": types.StringP(im.ctx.Name),
		},
	}

	_, err := im.conn.publicIPAddressesClient.CreateOrUpdate(im.namer.ResourceGroupName(), name, req, nil)
	if err != nil {
		return network.PublicIPAddress{}, err
	}
	im.ctx.Logger.Infof("Public ip addres %v created", name)
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
	storageName := im.ctx.AzureCloudConfig.StorageAccountName
	account, err := im.conn.storageClient.GetProperties(im.namer.ResourceGroupName(), storageName)
	return account, err
}

func (im *instanceManager) createNetworkInterface(name string, subnet network.Subnet, alloc network.IPAllocationMethod, internalIP string, pip network.PublicIPAddress) (network.Interface, error) {
	req := network.Interface{
		Name:     types.StringP(name),
		Location: types.StringP(im.ctx.Zone),
		InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
			IPConfigurations: &[]network.InterfaceIPConfiguration{
				{
					Name: types.StringP("ipconfig"),
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
			EnableIPForwarding: types.TrueP(),
		},
		Tags: &map[string]*string{
			"KubernetesCluster": types.StringP(im.ctx.Name),
		},
	}
	if alloc == network.Static {
		if internalIP == "" {
			return network.Interface{}, errors.New("No private IP provided for Static allocation.").WithContext(im.ctx).Err()
		}
		(*req.IPConfigurations)[0].PrivateIPAddress = types.StringP(internalIP)
	}
	_, err := im.conn.interfacesClient.CreateOrUpdate(im.namer.ResourceGroupName(), name, req, nil)
	if err != nil {
		return network.Interface{}, err
	}
	im.ctx.Logger.Infof("Network interface %v created", name)
	return im.conn.interfacesClient.Get(im.namer.ResourceGroupName(), name, "")
}

func (im *instanceManager) createVirtualMachine(nic network.Interface, as compute.AvailabilitySet, sa armstorage.Account, vmName, data, vmSize string) (compute.VirtualMachine, error) {
	req := compute.VirtualMachine{
		Name:     types.StringP(vmName),
		Location: types.StringP(im.ctx.Zone),
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
				ComputerName:  types.StringP(vmName),
				AdminPassword: types.StringP(im.ctx.InstanceRootPassword),
				AdminUsername: types.StringP(im.namer.AdminUsername()),
				CustomData:    types.StringP(base64.StdEncoding.EncodeToString([]byte(data))),
				LinuxConfiguration: &compute.LinuxConfiguration{
					DisablePasswordAuthentication: types.BoolP(!_env.FromHost().DebugEnabled()),
					SSH: &compute.SSHConfiguration{
						PublicKeys: &[]compute.SSHPublicKey{
							{
								KeyData: types.StringP(string(im.ctx.SSHKey.PublicKey)),
								Path:    types.StringP(fmt.Sprintf("/home/%v/.ssh/authorized_keys", im.namer.AdminUsername())),
							},
						},
					},
				},
			},
			StorageProfile: &compute.StorageProfile{
				ImageReference: &compute.ImageReference{
					Publisher: types.StringP(im.ctx.InstanceImageProject),
					Offer:     types.StringP(im.ctx.OS),
					Sku:       types.StringP(im.ctx.InstanceImage),
					Version:   types.StringP(im.ctx.InstanceImageVersion),
				},
				OsDisk: &compute.OSDisk{
					Caching:      compute.ReadWrite,
					CreateOption: compute.FromImage,
					Name:         types.StringP(im.namer.BootDiskName(vmName)),
					Vhd: &compute.VirtualHardDisk{
						URI: types.StringP(im.namer.BootDiskURI(sa, vmName)),
					},
				},
			},
			HardwareProfile: &compute.HardwareProfile{
				VMSize: compute.VirtualMachineSizeTypes(vmSize),
			},
		},
		Tags: &map[string]*string{
			"KubernetesCluster": types.StringP(im.ctx.Name),
		},
	}

	_, err := im.conn.vmClient.CreateOrUpdate(im.namer.ResourceGroupName(), vmName, req, nil)
	if err != nil {
		return compute.VirtualMachine{}, err
	}
	im.ctx.Logger.Infof("Virtual machine with disk %v password %v created", im.namer.BootDiskURI(sa, vmName), im.ctx.InstanceRootPassword)
	// https://docs.microsoft.com/en-us/azure/virtual-machines/virtual-machines-linux-extensions-customscript?toc=%2fazure%2fvirtual-machines%2flinux%2ftoc.json
	// https://github.com/Azure/custom-script-extension-linux
	// old: https://github.com/Azure/azure-linux-extensions/tree/master/CustomScript
	// https://docs.microsoft.com/en-us/azure/virtual-machines/virtual-machines-windows-classic-inject-custom-data
	im.ctx.Logger.Infof("Running startup script in virtual machine %v", vmName)
	extName := vmName + "-script"
	extReq := compute.VirtualMachineExtension{
		Name:     types.StringP(extName),
		Type:     types.StringP("Microsoft.Compute/virtualMachines/extensions"),
		Location: types.StringP(im.ctx.Zone),
		VirtualMachineExtensionProperties: &compute.VirtualMachineExtensionProperties{
			Publisher:               types.StringP("Microsoft.Azure.Extensions"),
			Type:                    types.StringP("CustomScript"),
			TypeHandlerVersion:      types.StringP("2.0"),
			AutoUpgradeMinorVersion: types.TrueP(),
			Settings: &map[string]interface{}{
				"commandToExecute": "cat /var/lib/waagent/CustomData | base64 --decode | /bin/bash",
			},
			// ProvisioningState
		},
		Tags: &map[string]*string{
			"KubernetesCluster": types.StringP(im.ctx.Name),
		},
	}
	_, err = im.conn.vmExtensionsClient.CreateOrUpdate(im.namer.ResourceGroupName(), vmName, extName, extReq, nil)
	if err != nil {
		return compute.VirtualMachine{}, err
	}

	im.ctx.Logger.Infof("Restarting virtual machine %v", vmName)
	_, err = im.conn.vmClient.Restart(im.namer.ResourceGroupName(), vmName, nil)
	if err != nil {
		return compute.VirtualMachine{}, err
	}

	vm, err := im.conn.vmClient.Get(im.namer.ResourceGroupName(), vmName, compute.InstanceView)
	im.ctx.Logger.Infof("Found virtual machine %v", vm)
	return vm, err
}

func (im *instanceManager) DeleteVirtualMachine(vmName string) error {
	_, err := im.conn.vmClient.Delete(im.namer.ResourceGroupName(), vmName, nil)
	storageName := im.ctx.AzureCloudConfig.StorageAccountName
	keys, err := im.conn.storageClient.ListKeys(im.namer.ResourceGroupName(), storageName)
	if err != nil {
		return err
	}
	im.ctx.Logger.Infof("Virtual machine %v deleted", vmName)
	storageClient, err := azstore.NewBasicClient(storageName, *(*(keys.Keys))[0].Value)
	if err != nil {
		return err
	}
	_, err = storageClient.GetBlobService().DeleteBlobIfExists(storageName, im.namer.BlobName(vmName), nil)
	if err != nil {
		return err
	}
	return nil
}

// http://askubuntu.com/questions/9853/how-can-i-make-rc-local-run-on-startup
func (im *instanceManager) RenderStartupScript(opt *contexts.ScriptOptions, sku, role string) string {
	cmd := lib.StartupConfigFromAPI(opt, role)
	if api.UseFirebase() {
		cmd = lib.StartupConfigFromFirebase(opt, role)
	}

	firebaseUid := ""
	if api.UseFirebase() {
		firebaseUid, _ = api.FirebaseUid()
	}
	return fmt.Sprintf(`#!/bin/bash
cat >/etc/kube-installer.sh <<EOF
%v
rm /lib/systemd/system/kube-installer.service
systemctl daemon-reload
exit 0
EOF
chmod +x /etc/kube-installer.sh

cat >/lib/systemd/system/kube-installer.service <<EOF
[Unit]
Description=Install Kubernetes Master

[Service]
Type=simple
Environment="APPSCODE_ENV=%v"
Environment="FIREBASE_UID=%v"

ExecStart=/bin/bash -e /etc/kube-installer.sh
Restart=on-failure
StartLimitInterval=5

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable kube-installer.service

/bin/sed -i 's/GRUB_CMDLINE_LINUX="/GRUB_CMDLINE_LINUX="cgroup_enable=memory swapaccount=1 /' /etc/default/grub
/usr/sbin/update-grub

# Don't restart inside script for Azure, call api to restart
# /sbin/reboot
`, strings.Replace(lib.RenderKubeStarter(opt, sku, cmd), "$", "\\$", -1), _env.FromHost().String(), firebaseUid)
}

func (im *instanceManager) newKubeInstance(vm compute.VirtualMachine, nic network.Interface, pip network.PublicIPAddress) (*contexts.KubernetesInstance, error) {
	i := contexts.KubernetesInstance{
		PHID:           phid.NewKubeInstance(),
		ExternalID:     fmt.Sprintf(machineIDTemplate, im.ctx.CloudCredential[credentialutil.AzureCredentialSubscriptionID], im.namer.ResourceGroupName(), *vm.Name),
		ExternalStatus: *vm.ProvisioningState,
		Name:           *vm.Name,
		InternalIP:     *(*nic.IPConfigurations)[0].PrivateIPAddress,
		SKU:            string(vm.HardwareProfile.VMSize),
		Status:         storage.KubernetesInstanceStatus_Ready,
	}
	if pip.IPAddress != nil {
		i.ExternalIP = *pip.IPAddress
	}
	return &i, nil
}
