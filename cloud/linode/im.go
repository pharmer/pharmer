package linode

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/appscode/data"
	"github.com/appscode/errors"
	_env "github.com/appscode/go/env"
	"github.com/appscode/linodego"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud/lib"
	"github.com/appscode/pharmer/phid"
	"github.com/appscode/pharmer/storage"
	"github.com/cenkalti/backoff"
)

type instanceManager struct {
	ctx   *api.Cluster
	conn  *cloudConnector
	namer namer
}

const (
	LinodeStatus_BeingCreated = -1
	LinodeStatus_BrandNew     = 0
	LinodeStatus_Running      = 1
	LinodeStatus_PoweredOff   = 2
)

func (im *instanceManager) GetInstance(md *api.InstanceMetadata) (*api.KubernetesInstance, error) {
	master := net.ParseIP(md.Name) == nil

	var instance *api.KubernetesInstance
	backoff.Retry(func() error {
		resp, err := im.conn.client.Ip.List(0, 0)
		if err != nil {
			return err
		}
		for _, fip := range resp.FullIPAddresses {
			if fip.IsPublic == 0 && fip.IPAddress == md.InternalIP {
				linodes, err := im.conn.client.Linode.List(fip.LinodeId)
				if err != nil {
					return err
				}
				instance, err = im.newKubeInstance(&linodes.Linodes[0])
				if err != nil {
					return err
				}
				if master {
					instance.Role = api.RoleKubernetesMaster
				} else {
					instance.Name = im.ctx.Name + "-node-" + strconv.Itoa(fip.LinodeId)
					instance.Role = api.RoleKubernetesPool
				}
				return nil
			}
		}
		return nil
	}, backoff.NewExponentialBackOff())

	if instance == nil {
		return nil, errors.New("No instance found with name", md.Name).WithContext(im.ctx).Err()
	}
	return instance, nil
}

func (im *instanceManager) createStackScript(sku, role string) (int, error) {
	startupScript := im.RenderStartupScript(im.ctx.NewScriptOptions(), sku, role)
	script, err := im.conn.client.StackScript.Create(im.namer.StartupScriptName(sku, role), im.ctx.InstanceImage, startupScript, map[string]string{
		"Description": im.ctx.Name,
	})
	if err != nil {
		return 0, err
	}
	im.ctx.Logger().Infof("Stack script for role %v created", role)
	return script.StackScriptId.StackScriptId, nil
}

// http://askubuntu.com/questions/9853/how-can-i-make-rc-local-run-on-startup
func (im *instanceManager) RenderStartupScript(opt *api.ScriptOptions, sku, role string) string {
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

# http://ask.xmodulo.com/disable-ipv6-linux.html
/bin/cat >>/etc/sysctl.conf <<EOF
# to disable IPv6 on all interfaces system wide
net.ipv6.conf.all.disable_ipv6 = 1

# to disable IPv6 on a specific interface (e.g., eth0, lo)
net.ipv6.conf.lo.disable_ipv6 = 1
net.ipv6.conf.eth0.disable_ipv6 = 1
EOF
/sbin/sysctl -p /etc/sysctl.conf
/bin/sed -i 's/^#AddressFamily any/AddressFamily inet/' /etc/ssh/sshd_config

export DEBIAN_FRONTEND=noninteractive
export DEBCONF_NONINTERACTIVE_SEEN=true
/usr/bin/apt-get update
/usr/bin/apt-get install -y --no-install-recommends --force-yes linux-image-amd64 grub2

/bin/cat >/etc/default/grub <<EOF
GRUB_DEFAULT=0
GRUB_TIMEOUT=10
GRUB_DISTRIBUTOR=\$(lsb_release -i -s 2> /dev/null || echo Debian)
GRUB_CMDLINE_LINUX_DEFAULT="quiet"
GRUB_CMDLINE_LINUX="cgroup_enable=memory swapaccount=1 console=ttyS0,19200n8"
GRUB_DISABLE_LINUX_UUID=true
GRUB_SERIAL_COMMAND="serial --speed=19200 --unit=0 --word=8 --parity=no --stop=1"
GRUB_TERMINAL=serial
EOF

/usr/sbin/update-grub
/sbin/poweroff
`, strings.Replace(lib.RenderKubeStarter(opt, sku, cmd), "$", "\\$", -1), _env.FromHost(), firebaseUid)
}

func (im *instanceManager) createInstance(name string, scriptId int, sku string) (int, int, error) {
	dcId, err := strconv.Atoi(im.ctx.Zone)
	if err != nil {
		return 0, 0, errors.FromErr(err).WithContext(im.ctx).Err()
	}
	planId, err := strconv.Atoi(sku)
	if err != nil {
		return 0, 0, errors.FromErr(err).WithContext(im.ctx).Err()
	}
	linode, err := im.conn.client.Linode.Create(dcId, planId, 0)
	if err != nil {
		return 0, 0, errors.FromErr(err).WithContext(im.ctx).Err()
	}
	id := linode.LinodeId.LinodeId

	_, err = im.conn.client.Ip.AddPrivate(id)
	if err != nil {
		return 0, 0, errors.FromErr(err).WithContext(im.ctx).Err()
	}

	_, err = im.conn.client.Linode.Update(id, map[string]interface{}{
		"watchdog": false,
	})
	if err != nil {
		return 0, 0, errors.FromErr(err).WithContext(im.ctx).Err()
	}

	stackScriptUDFResponses := fmt.Sprintf(`{
  "cluster": "%v",
  "instance": "%v",
  "stack_script_id": "%v"
}`, im.ctx.Name, name, scriptId)
	args := map[string]string{
		"rootSSHKey": string(im.ctx.SSHKey.PublicKey),
	}

	mt, err := data.ClusterMachineType("linode", sku)
	if err != nil {
		return 0, 0, errors.FromErr(err).WithContext(im.ctx).Err()
	}
	distributionID, err := strconv.Atoi(im.ctx.InstanceImage)
	if err != nil {
		return 0, 0, errors.FromErr(err).WithContext(im.ctx).Err()
	}
	swapDiskSize := 512                // MB
	rootDiskSize := mt.Disk*1024 - 512 // MB
	rootDisk, err := im.conn.client.Disk.CreateFromStackscript(scriptId, id, name, stackScriptUDFResponses, distributionID, rootDiskSize, im.ctx.InstanceRootPassword, args)
	if err != nil {
		return 0, 0, errors.FromErr(err).WithContext(im.ctx).Err()
	}
	swapDisk, err := im.conn.client.Disk.Create(id, "swap", "swap-disk", swapDiskSize, nil)
	if err != nil {
		return 0, 0, errors.FromErr(err).WithContext(im.ctx).Err()
	}

	kernelId, err := strconv.Atoi(im.ctx.Kernel)
	if err != nil {
		return 0, 0, errors.FromErr(err).WithContext(im.ctx).Err()
	}
	args = map[string]string{
		"RootDeviceNum": "1",
		"DiskList":      fmt.Sprintf("%d,%d", rootDisk.DiskJob.DiskId, swapDisk.DiskJob.DiskId),
	}
	config, err := im.conn.client.Config.Create(id, kernelId, name, args)
	if err != nil {
		return 0, 0, errors.FromErr(err).WithContext(im.ctx).Err()
	}

	jobResp, err := im.conn.client.Linode.Boot(id, config.LinodeConfigId.LinodeConfigId)
	if err != nil {
		return 0, 0, errors.FromErr(err).WithContext(im.ctx).Err()
	}
	im.ctx.Logger().Info("Running linode boot job %v", jobResp.JobId.JobId)
	im.ctx.Logger().Infof("Linode %v created", name)

	return id, config.LinodeConfigId.LinodeConfigId, err
}

func (im *instanceManager) bootToGrub2(linodeId, configId int, name string) error {
	// GRUB2 Kernel ID = 210
	_, err := im.conn.client.Config.Update(configId, linodeId, 210, nil)
	if err != nil {
		return err
	}
	_, err = im.conn.client.Linode.Update(linodeId, map[string]interface{}{
		"Label":    name,
		"watchdog": true,
	})
	if err != nil {
		return err
	}
	_, err = im.conn.client.Linode.Boot(linodeId, configId)
	im.ctx.Logger().Infof("%v booted", name)
	return err
}

func (im *instanceManager) newKubeInstance(linode *linodego.Linode) (*api.KubernetesInstance, error) {
	var externalIP, internalIP string
	ips, err := im.conn.client.Ip.List(linode.LinodeId, -1)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(im.ctx).Err()
	}
	for _, ip := range ips.FullIPAddresses {
		if ip.IsPublic == 1 {
			externalIP = ip.IPAddress
		} else {
			internalIP = ip.IPAddress
		}
		if externalIP != "" && internalIP != "" {
			i := api.KubernetesInstance{
				PHID:           phid.NewKubeInstance(),
				ExternalID:     strconv.Itoa(linode.LinodeId),
				Name:           linode.Label.String(),
				ExternalIP:     externalIP,
				InternalIP:     internalIP,
				SKU:            strconv.Itoa(linode.PlanId),
				Status:         storage.KubernetesInstanceStatus_Ready,
				ExternalStatus: statusString(linode.Status),
			}
			return &i, nil
		}
	}
	return nil, errors.New("Failed to detect Public IP").WithContext(im.ctx).Err()
}
