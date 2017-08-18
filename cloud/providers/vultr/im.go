package vultr

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	gv "github.com/JamesClonk/vultr/lib"
	"github.com/appscode/errors"
	_env "github.com/appscode/go/env"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/context"
	"github.com/appscode/pharmer/phid"
	"github.com/cenkalti/backoff"
)

type instanceManager struct {
	ctx     context.Context
	cluster *api.Cluster
	conn    *cloudConnector
	namer   namer
}

func (im *instanceManager) GetInstance(md *api.InstanceMetadata) (*api.KubernetesInstance, error) {
	master := net.ParseIP(md.Name) == nil

	var instance *api.KubernetesInstance
	backoff.Retry(func() (err error) {
		servers, err := im.conn.client.GetServers()
		if err != nil {
			return
		}
		for _, server := range servers {
			if server.InternalIP == md.InternalIP {
				instance, err = im.newKubeInstance(&server)
				if master {
					instance.Role = api.RoleKubernetesMaster
				} else {
					instance.Role = api.RoleKubernetesPool
				}
				return
			}
		}
		return nil
	}, backoff.NewExponentialBackOff())

	if instance == nil {
		return nil, errors.New("No instance found with name", md.Name).WithContext(im.ctx).Err()
	}
	return instance, nil
}

func (im *instanceManager) createStartupScript(sku, role string) (int, error) {
	im.ctx.Logger().Infof("creating StackScript for sku %v role %v", sku, role)
	script := im.RenderStartupScript(sku, role)

	resp, err := im.conn.client.CreateStartupScript(im.namer.StartupScriptName(sku, role), script, "boot")
	if err != nil {
		return 0, err
	}
	scriptID, err := strconv.Atoi(resp.ID)
	if err != nil {
		return 0, errors.FromErr(err).WithContext(im.ctx).Err()
	}
	return scriptID, nil
}

// http://askubuntu.com/questions/9853/how-can-i-make-rc-local-run-on-startup
func (im *instanceManager) RenderStartupScript(sku, role string) string {
	cmd := cloud.StartupConfigFromAPI(im.cluster, role)
	if api.UseFirebase() {
		cmd = cloud.StartupConfigFromFirebase(im.cluster, role)
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
ExecStart=/bin/bash -e /etc/kube-installer.sh
Restart=on-failure
StartLimitInterval=5

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable kube-installer.service

# https://www.vultr.com/docs/configuring-private-network
PRIVATE_ADDRESS=$(/usr/bin/curl http://169.254.169.254/v1/interfaces/1/ipv4/address 2> /dev/null)
PRIVATE_NETMASK=$(/usr/bin/curl http://169.254.169.254/v1/interfaces/1/ipv4/netmask 2> /dev/null)
/bin/cat >>/etc/network/interfaces <<EOF

auto eth1
iface eth1 inet static
    address $PRIVATE_ADDRESS
    netmask $PRIVATE_NETMASK
            mtu 1450
EOF

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

`, strings.Replace(cloud.RenderKubeStarter(im.cluster, sku, cmd), "$", "\\$", -1))
}

func (im *instanceManager) createInstance(name, sku string, scriptID int) (string, error) {
	regionID, err := strconv.Atoi(im.cluster.Spec.Zone)
	if err != nil {
		return "", errors.FromErr(err).WithContext(im.ctx).Err()
	}
	planID, err := strconv.Atoi(sku)
	if err != nil {
		return "", errors.FromErr(err).WithContext(im.ctx).Err()
	}
	osID, err := strconv.Atoi(im.cluster.Spec.InstanceImage)
	if err != nil {
		return "", errors.FromErr(err).WithContext(im.ctx).Err()
	}
	opts := &gv.ServerOptions{
		SSHKey:               im.cluster.Spec.SSHKeyExternalID + ",57dcbce7cd3b6,58027d56a1190,58a498ec7ee19",
		PrivateNetworking:    true,
		DontNotifyOnActivate: false,
		Script:               scriptID,
		Hostname:             name,
		Tag:                  im.cluster.Name,
	}
	if _env.FromHost().IsPublic() {
		opts.SSHKey = im.cluster.Spec.SSHKeyExternalID
	}
	resp, err := im.conn.client.CreateServer(
		name,
		regionID,
		planID,
		osID,
		opts)
	im.ctx.Logger().Debugln("do response", resp, " errors", err)
	im.ctx.Logger().Debug("Created droplet with name", resp.ID)
	im.ctx.Logger().Infof("DO droplet %v created", name)
	return resp.ID, err
}

func (im *instanceManager) assignReservedIP(ip, serverId string) error {
	err := im.conn.client.AttachReservedIP(ip, serverId)
	if err != nil {
		return errors.FromErr(err).WithContext(im.ctx).Err()
	}
	im.ctx.Logger().Infof("Reserved ip %v assigned to %v", ip, serverId)
	return nil
}

func (im *instanceManager) newKubeInstance(server *gv.Server) (*api.KubernetesInstance, error) {
	return &api.KubernetesInstance{
		PHID:           phid.NewKubeInstance(),
		ExternalID:     server.ID,
		ExternalStatus: server.Status + "|" + server.PowerStatus,
		Name:           server.Name,
		ExternalIP:     server.MainIP,
		InternalIP:     server.InternalIP,
		SKU:            strconv.Itoa(server.PlanID),        // 512mb // convert to SKU
		Status:         api.KubernetesInstanceStatus_Ready, // active
	}, nil
}

// reboot does not seem to run /etc/rc.local
func (im *instanceManager) reboot(id string) error {
	im.ctx.Logger().Infof("Rebooting instance %v", id)
	err := im.conn.client.RebootServer(id)
	if err != nil {
		return errors.FromErr(err).WithContext(im.ctx).Err()
	}
	return nil
}
