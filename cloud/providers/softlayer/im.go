package softlayer

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/appscode/data"
	"github.com/appscode/errors"
	_env "github.com/appscode/go/env"
	"github.com/appscode/go/types"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/context"
	"github.com/appscode/pharmer/phid"
	"github.com/cenkalti/backoff"
	"github.com/softlayer/softlayer-go/datatypes"
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
			servers, err := im.conn.accountServiceClient.GetVirtualGuests()
			if err != nil {
				return err
			}
			for _, s := range servers {
				interIp := strings.Trim(*s.PrimaryBackendIpAddress, `"`)
				if interIp == md.InternalIP {
					instance, err = im.newKubeInstance(*s.Id)
					sku := strconv.Itoa(*s.MaxCpu) + "c" + strconv.Itoa(*s.MaxMemory) + "m"
					instance.Spec.SKU = sku
					if err != nil {
						return err
					}
					if master {
						instance.Spec.Role = api.RoleKubernetesMaster
					} else {
						instance.Spec.Role = api.RoleKubernetesPool
					}
					return nil
				}

			}
		}
		return nil
	}, backoff.NewExponentialBackOff())

	if instance == nil {
		return nil, errors.New("No instance found with name", md.Name).WithContext(im.ctx).Err()
	}
	return instance, nil
}

func (im *instanceManager) createInstance(name, role, sku string) (int, error) {
	startupScript := im.RenderStartupScript(sku, role)
	instance, err := data.ClusterMachineType(im.cluster.Spec.Provider, sku)
	if err != nil {
		im.cluster.Status.Reason = err.Error()
		return 0, errors.FromErr(err).WithContext(im.ctx).Err()
	}
	cpu := instance.CPU
	ram := 0
	switch instance.RAM.(type) {
	case int, int32, int64:
		ram = instance.RAM.(int) * 1024
	case float64, float32:
		ram = int(instance.RAM.(float64) * 1024)
	default:
		return 0, fmt.Errorf("Failed to parse memory metadata for sku %v", sku)
	}

	sshid, err := strconv.Atoi(im.cluster.Spec.SSHKeyExternalID)
	if err != nil {
		im.cluster.Status.Reason = err.Error()
		return 0, errors.FromErr(err).WithContext(im.ctx).Err()
	}
	vGuestTemplate := datatypes.Virtual_Guest{
		Hostname:                     types.StringP(name),
		Domain:                       types.StringP(im.ctx.Extra().ExternalDomain(im.cluster.Name)),
		MaxMemory:                    types.IntP(ram),
		StartCpus:                    types.IntP(cpu),
		Datacenter:                   &datatypes.Location{Name: types.StringP(im.cluster.Spec.Zone)},
		OperatingSystemReferenceCode: types.StringP(im.cluster.Spec.OS),
		LocalDiskFlag:                types.TrueP(),
		HourlyBillingFlag:            types.TrueP(),
		SshKeys: []datatypes.Security_Ssh_Key{
			{
				Id:          types.IntP(sshid),
				Fingerprint: types.StringP(im.cluster.Spec.SSHKey.OpensshFingerprint),
			},
		},
		UserData: []datatypes.Virtual_Guest_Attribute{
			{
				//https://sldn.softlayer.com/blog/jarteche/getting-started-user-data-and-post-provisioning-scripts
				Type: &datatypes.Virtual_Guest_Attribute_Type{
					Keyname: types.StringP("USER_DATA"),
					Name:    types.StringP("User Data"),
				},
				Value: types.StringP(startupScript),
			},
		},
		PostInstallScriptUri: types.StringP("https://raw.githubusercontent.com/appscode/pharmer/master/cloud/providers/softlayer/startupscript.sh"),
	}

	vGuest, err := im.conn.virtualServiceClient.Mask("id;domain").CreateObject(&vGuestTemplate)
	if err != nil {
		im.cluster.Status.Reason = err.Error()
		return 0, errors.FromErr(err).WithContext(im.ctx).Err()
	}
	im.ctx.Logger().Infof("Softlayer instance %v created", name)
	return *vGuest.Id, nil
}

func (im *instanceManager) RenderStartupScript(sku, role string) string {
	cmd := cloud.StartupConfigFromAPI(im.cluster, role)
	if api.UseFirebase() {
		cmd = cloud.StartupConfigFromFirebase(im.cluster, role)
	}

	firebaseUid := ""
	if api.UseFirebase() {
		firebaseUid, _ = api.FirebaseUid()
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

%v
`, strings.Replace(cloud.RenderKubeStarter(im.cluster, sku, cmd), "$", "\\$", -1), _env.FromHost().String(), firebaseUid, reboot)
}

func (im *instanceManager) newKubeInstance(id int) (*api.Instance, error) {
	bluemix := im.conn.virtualServiceClient.Id(id)
	status, err := bluemix.GetStatus()
	if err != nil {
		return nil, errors.FromErr(err).WithContext(im.ctx).Err()
	}
	d, err := bluemix.GetObject()
	if err != nil {
		return nil, errors.FromErr(err).WithContext(im.ctx).Err()
	}
	ki := &api.Instance{
		ObjectMeta: api.ObjectMeta{
			UID:  phid.NewKubeInstance(),
			Name: *d.FullyQualifiedDomainName,
		},
		Status: api.InstanceStatus{
			ExternalID:    strconv.Itoa(id),
			ExternalPhase: *status.Name,
			Phase:         api.InstancePhaseReady, // droplet.Status == active
		},
	}

	ki.Status.ExternalIP, err = bluemix.GetPrimaryIpAddress()
	if err != nil {
		return nil, errors.FromErr(err).WithContext(im.ctx).Err()
	}
	ki.Status.ExternalIP = strings.Trim(ki.Status.ExternalIP, `"`)
	ki.Status.InternalIP, err = bluemix.GetPrimaryBackendIpAddress()
	if err != nil {
		return nil, errors.FromErr(err).WithContext(im.ctx).Err()
	}
	ki.Status.InternalIP = strings.Trim(ki.Status.InternalIP, `"`)

	return ki, nil
}

func (im *instanceManager) reboot(id int) (bool, error) {
	service := im.conn.virtualServiceClient.Id(id)
	return service.RebootDefault()
}
