package packet

import (
	"fmt"
	"net"
	"strings"

	"github.com/appscode/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/common"
	"github.com/appscode/pharmer/contexts"
	"github.com/appscode/pharmer/phid"
	"github.com/appscode/pharmer/storage"
	"github.com/appscode/pharmer/system"
	"github.com/cenkalti/backoff"
	"github.com/packethost/packngo"
)

type instanceManager struct {
	ctx  *contexts.ClusterContext
	conn *cloudConnector
}

func (im *instanceManager) GetInstance(md *contexts.InstanceMetadata) (*contexts.KubernetesInstance, error) {
	master := net.ParseIP(md.Name) == nil

	var instance *contexts.KubernetesInstance
	backoff.Retry(func() (err error) {
		for {
			var servers []packngo.Device
			servers, _, err = im.conn.client.Devices.List(im.ctx.Project)
			if err != nil {
				return
			}
			for _, s := range servers {
				for _, ipAddr := range s.Network {
					if ipAddr.AddressFamily == 4 && ipAddr.Public && ipAddr.Address == md.InternalIP {
						instance, err = im.newKubeInstanceFromServer(&s)
						if err != nil {
							return
						}
						if master {
							instance.Role = system.RoleKubernetesMaster
						} else {
							instance.Role = system.RoleKubernetesPool
						}
						return
					}
				}
			}
		}
		return
	}, backoff.NewExponentialBackOff())

	if instance == nil {
		return nil, errors.New("No instance found with name", md.Name).WithContext(im.ctx).Err()
	}
	return instance, nil
}

func (im *instanceManager) createInstance(name, role, sku string, ipid ...string) (*packngo.Device, error) {
	startupScript := im.RenderStartupScript(im.ctx.NewScriptOptions(), sku, role)
	device, _, err := im.conn.client.Devices.Create(&packngo.DeviceCreateRequest{
		HostName:     name,
		Plan:         sku,
		Facility:     im.ctx.Zone,
		OS:           im.ctx.InstanceImage,
		BillingCycle: "hourly",
		ProjectID:    im.ctx.Project,
		UserData:     startupScript,
		Tags:         []string{im.ctx.Name},
	})
	im.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, fmt.Sprintf("Instance %v created", name))
	return device, err
}

// http://askubuntu.com/questions/9853/how-can-i-make-rc-local-run-on-startup
func (im *instanceManager) RenderStartupScript(opt *contexts.ScriptOptions, sku, role string) string {
	cmd := common.StartupConfigFromAPI(opt, role)
	if api.UseFirebase() {
		cmd = common.StartupConfigFromFirebase(opt, role)
	}

	reboot := ""
	if role == system.RoleKubernetesPool {
		reboot = "/sbin/reboot"
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
Environment=DEBIAN_FRONTEND=noninteractive
Environment=DEBCONF_NONINTERACTIVE_SEEN=true
ExecStartPre=/usr/bin/apt-get -y -f dist-upgrade
ExecStart=/bin/bash -e /etc/kube-installer.sh
Restart=on-failure
StartLimitInterval=5

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable kube-installer.service

/bin/cat >/etc/apt/sources.list <<EOF
deb http://ftp.us.debian.org/debian jessie main
deb http://security.debian.org/ jessie/updates main
deb http://ftp.us.debian.org/debian jessie-updates main
EOF
/usr/bin/apt-get update

/bin/sed -i 's/\/boot\/vmlinuz/\/boot\/vmlinuz\ cgroup_enable=memory\ swapaccount=1/' /boot/grub/grub.cfg

%v
`, strings.Replace(common.RenderKubeStarter(opt, sku, cmd), "$", "\\$", -1), reboot)
}

func (im *instanceManager) newKubeInstance(id string) (*contexts.KubernetesInstance, error) {
	s, _, err := im.conn.client.Devices.Get(id)
	if err != nil {
		return nil, common.InstanceNotFound
	}
	return im.newKubeInstanceFromServer(s)
}

func (im *instanceManager) newKubeInstanceFromServer(droplet *packngo.Device) (*contexts.KubernetesInstance, error) {
	ki := &contexts.KubernetesInstance{
		PHID:           phid.NewKubeInstance(),
		ExternalID:     droplet.ID,
		ExternalStatus: droplet.State,
		Name:           droplet.Hostname,
		// ExternalIP:     droplet.PublicAddress.IP,
		// InternalIP:     droplet.PrivateIP,
		SKU:    droplet.Plan.ID,
		Status: storage.KubernetesInstanceStatus_Ready, // droplet.Status == active
	}
	for _, addr := range droplet.Network {
		if addr.AddressFamily == 4 {
			if addr.Public {
				ki.ExternalIP = addr.Address
			} else {
				ki.InternalIP = addr.Address
			}
		}
	}
	return ki, nil
}

// reboot does not seem to run /etc/rc.local
func (im *instanceManager) reboot(id string) error {
	im.ctx.Logger().Infof("Rebooting instance %v", id)
	_, err := im.conn.client.Devices.Reboot(id)
	if err != nil {
		return errors.FromErr(err).WithContext(im.ctx).Err()
	}
	return nil
}
