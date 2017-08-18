package hetzner

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/appscode/errors"
	hc "github.com/appscode/go-hetzner"
	_ssh "github.com/appscode/go/crypto/ssh"
	_env "github.com/appscode/go/env"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/context"
	"github.com/appscode/pharmer/phid"
	"golang.org/x/crypto/ssh"
)

type instanceManager struct {
	ctx     context.Context
	cluster *api.Cluster
	conn    *cloudConnector
}

func (im *instanceManager) GetInstance(md *api.InstanceMetadata) (*api.KubernetesInstance, error) {
	master := net.ParseIP(md.Name) == nil
	servers, _, err := im.conn.client.Server.ListServers()
	if err != nil {
		return nil, err
	}
	for _, s := range servers {
		if !s.Cancelled && s.ServerIP == md.ExternalIP {
			instance, err := im.newKubeInstanceFromSummary(s)
			if err != nil {
				return nil, err
			}
			if master {
				instance.Role = api.RoleKubernetesMaster
			} else {
				instance.Role = api.RoleKubernetesPool
			}
			return instance, nil

		}
	}
	return nil, errors.New("No instance found with name", md.Name).WithContext(im.ctx).Err()
}

func (im *instanceManager) createInstance(role, sku string) (*hc.Transaction, error) {
	tx, _, err := im.conn.client.Ordering.CreateTransaction(&hc.CreateTransactionRequest{
		ProductID:     sku,
		AuthorizedKey: []string{im.cluster.Spec.SSHKey.OpensshFingerprint},
		Dist:          im.cluster.Spec.InstanceImage,
		Arch:          64,
		Lang:          "en",
		// Test:          true,
	})
	im.ctx.Logger().Infof("Instance with sku %v created", sku)
	return tx, err
}

func (im *instanceManager) storeConfigFile(serverIP, role string, signer ssh.Signer) error {
	im.ctx.Logger().Infof("Storing config file for server %v", serverIP)
	cfg, err := im.cluster.StartupConfigResponse(role)
	if err != nil {
		return errors.FromErr(err).WithContext(im.ctx).Err()
	}
	fmt.Println(">>>>>>>>>>>>>>>>>>>>>>>", cfg)

	file := fmt.Sprintf("/var/cache/kubernetes_context_%v_%v.yaml", im.cluster.Spec.ResourceVersion, role)
	stdOut, stdErr, code, err := _ssh.SCP(file, []byte(cfg), "root", serverIP+":22", signer)
	im.ctx.Logger().Debugf(stdOut, stdErr, code)
	return err
}

func (im *instanceManager) storeStartupScript(serverIP, sku, role string, signer ssh.Signer) error {
	im.ctx.Logger().Infof("Storing startup script for server %v", serverIP)
	startupScript := im.RenderStartupScript(sku, role)
	fmt.Println(">>>>>>>>>>>>>>>>>>>>>>>", startupScript)

	file := "/var/cache/kubernetes_startupscript.sh"
	stdOut, stdErr, code, err := _ssh.SCP(file, []byte(startupScript), "root", serverIP+":22", signer)
	im.ctx.Logger().Debugf(stdOut, stdErr, code)
	return err
}

// http://askubuntu.com/questions/9853/how-can-i-make-rc-local-run-on-startup
func (im *instanceManager) RenderStartupScript(sku, role string) string {
	cmd := fmt.Sprintf(`CONFIG=$(cat /var/cache/kubernetes_context_%v_%v.yaml)`, im.cluster.Spec.ResourceVersion, role)
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

/bin/sed -i 's/GRUB_CMDLINE_LINUX=""/GRUB_CMDLINE_LINUX="cgroup_enable=memory swapaccount=1"/' /etc/default/grub
/usr/sbin/update-grub

/sbin/reboot
`, strings.Replace(cloud.RenderKubeStarter(im.cluster, sku, cmd), "$", "\\$", -1), _env.FromHost().String(), firebaseUid)
}

func (im *instanceManager) executeStartupScript(serverIP string, signer ssh.Signer) error {
	im.ctx.Logger().Infof("SSH execing start command %v", serverIP+":22")

	stdOut, stdErr, code, err := _ssh.Exec(`bash /var/cache/kubernetes_startupscript.sh`, "root", serverIP+":22", signer)
	im.ctx.Logger().Debugf(stdOut, stdErr, code)
	if err != nil {
		return errors.FromErr(err).WithContext(im.ctx).Err()
	}
	return nil
}

func (im *instanceManager) newKubeInstance(serverIP string) (*api.KubernetesInstance, error) {
	s, _, err := im.conn.client.Server.GetServer(serverIP)
	if err != nil {
		return nil, cloud.InstanceNotFound
	}
	return im.newKubeInstanceFromSummary(&s.ServerSummary)
}

func (im *instanceManager) newKubeInstanceFromSummary(droplet *hc.ServerSummary) (*api.KubernetesInstance, error) {
	return &api.KubernetesInstance{
		PHID:           phid.NewKubeInstance(),
		ExternalID:     strconv.Itoa(droplet.ServerNumber),
		ExternalStatus: droplet.Status,
		Name:           droplet.ServerName,
		ExternalIP:     droplet.ServerIP,
		InternalIP:     "",
		SKU:            droplet.Product,
		Status:         api.KubernetesInstanceStatus_Ready, // droplet.Status == active
	}, nil
}
