package packet

import (
	"context"
	"net"

	"github.com/appscode/go/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/phid"
	"github.com/cenkalti/backoff"
	"github.com/packethost/packngo"
)

type instanceManager struct {
	ctx     context.Context
	cluster *api.Cluster
	conn    *cloudConnector
}

func (im *instanceManager) GetInstance(md *api.InstanceStatus) (*api.Instance, error) {
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
					if ipAddr.AddressFamily == 4 && ipAddr.Public && ipAddr.Address == md.PrivateIP {
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
	startupScript, err := RenderStartupScript(im.ctx, im.cluster, role)
	if err != nil {
		return nil, err
	}
	device, _, err := im.conn.client.Devices.Create(&packngo.DeviceCreateRequest{
		Hostname:     name,
		Plan:         sku,
		Facility:     im.cluster.Spec.Zone,
		OS:           im.cluster.Spec.InstanceImage,
		BillingCycle: "hourly",
		ProjectID:    im.cluster.Spec.Project,
		UserData:     startupScript,
		Tags:         []string{im.cluster.Name},
	})
	cloud.Logger(im.ctx).Infof("Instance %v created", name)
	return device, err
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
				ki.Status.PublicIP = addr.Address
			} else {
				ki.Status.PrivateIP = addr.Address
			}
		}
	}
	return ki, nil
}

// reboot does not seem to run /etc/rc.local
func (im *instanceManager) reboot(id string) error {
	cloud.Logger(im.ctx).Infof("Rebooting instance %v", id)
	_, err := im.conn.client.Devices.Reboot(id)
	if err != nil {
		return errors.FromErr(err).WithContext(im.ctx).Err()
	}
	return nil
}
