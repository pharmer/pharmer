package packet

import (
	"fmt"
	"net"
	"strings"

	"github.com/appscode/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/context"
	"github.com/appscode/pharmer/phid"
	"github.com/cenkalti/backoff"
	"github.com/packethost/packngo"
)

type instanceManager struct {
	ctx     context.Context
	cluster *api.Cluster
	conn    *cloudConnector
}

func (im *instanceManager) GetInstance(md *api.InstanceMetadata) (*api.Instance, error) {
	master := net.ParseIP(md.Name) == nil

	var instance *api.Instance
	backoff.Retry(func() (err error) {
		for {
			var servers []packngo.Device
			servers, _, err = im.conn.client.Devices.List(im.cluster.Spec.Project)
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
							instance.Spec.Role = api.RoleKubernetesMaster
						} else {
							instance.Spec.Role = api.RoleKubernetesPool
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
	startupScript := im.RenderStartupScript(sku, role)
	device, _, err := im.conn.client.Devices.Create(&packngo.DeviceCreateRequest{
		HostName:     name,
		Plan:         sku,
		Facility:     im.cluster.Spec.Zone,
		OS:           im.cluster.Spec.InstanceImage,
		BillingCycle: "hourly",
		ProjectID:    im.cluster.Spec.Project,
		UserData:     startupScript,
		Tags:         []string{im.cluster.Name},
	})
	im.ctx.Logger().Infof("Instance %v created", name)
	return device, err
}

// http://askubuntu.com/questions/9853/how-can-i-make-rc-local-run-on-startup
func (im *instanceManager) RenderStartupScript(sku, role string) string {
	cmd := cloud.StartupConfigFromAPI(im.cluster, role)
	if api.UseFirebase() {
		cmd = cloud.StartupConfigFromFirebase(im.cluster, role)
	}

	reboot := ""
	if role == api.RoleKubernetesPool {
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
`, strings.Replace(cloud.RenderKubeStarter(im.cluster, sku, cmd), "$", "\\$", -1), reboot)
}

func (im *instanceManager) newKubeInstance(id string) (*api.Instance, error) {
	s, _, err := im.conn.client.Devices.Get(id)
	if err != nil {
		return nil, cloud.InstanceNotFound
	}
	return im.newKubeInstanceFromServer(s)
}

func (im *instanceManager) newKubeInstanceFromServer(droplet *packngo.Device) (*api.Instance, error) {
	ki := &api.Instance{
		ObjectMeta: api.ObjectMeta{
			UID:  phid.NewKubeInstance(),
			Name: droplet.Hostname,
		},
		Spec: api.InstanceSpec{
			SKU: droplet.Plan.ID,
		},
		Status: api.InstanceStatus{
			// ExternalIP:     droplet.PublicAddress.IP,
			// InternalIP:     droplet.PrivateIP,
			ExternalID:    droplet.ID,
			ExternalPhase: droplet.State,
			Phase:         api.InstancePhaseReady, // droplet.Status == active
		},
	}
	for _, addr := range droplet.Network {
		if addr.AddressFamily == 4 {
			if addr.Public {
				ki.Status.ExternalIP = addr.Address
			} else {
				ki.Status.InternalIP = addr.Address
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
