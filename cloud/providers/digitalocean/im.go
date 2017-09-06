package digitalocean

import (
	"context"
	gtx "context"
	"fmt"
	"net"
	"strconv"

	_env "github.com/appscode/go/env"
	"github.com/appscode/go/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/phid"
	"github.com/cenkalti/backoff"
	"github.com/digitalocean/godo"
	"github.com/tamalsaha/go-oneliners"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type instanceManager struct {
	ctx     context.Context
	cluster *api.Cluster
	conn    *cloudConnector
	namer   namer
}

const DROPLET_IMAGE_SLUG = "ubuntu-16-04-x64"

func (im *instanceManager) GetInstance(md *api.InstanceStatus) (*api.Instance, error) {
	master := net.ParseIP(md.Name) == nil

	var instance *api.Instance
	backoff.Retry(func() (err error) {
		const pageSize = 50
		curPage := 0
		for {
			var droplets []godo.Droplet
			droplets, _, err = im.conn.client.Droplets.List(gtx.TODO(), &godo.ListOptions{
				Page:    curPage,
				PerPage: pageSize,
			})
			if err != nil {
				return
			}
			for _, droplet := range droplets {
				var internalIP string
				internalIP, err = droplet.PrivateIPv4()
				if err != nil {
					return
				}
				if internalIP == md.PrivateIP {
					instance, err = im.newKubeInstanceFromDroplet(&droplet)
					if master {
						instance.Spec.Role = api.RoleKubernetesMaster
					} else {
						instance.Spec.Role = api.RoleKubernetesPool
					}
					return
				}
			}
			curPage++
			if len(droplets) < pageSize {
				break
			}
		}
		return nil
	}, backoff.NewExponentialBackOff())

	if instance == nil {
		return nil, errors.New("No instance found with name", md.Name).WithContext(im.ctx).Err()
	}
	return instance, nil
}

func (im *instanceManager) createInstance(name, role, sku string) (*godo.Droplet, error) {
	oneliners.FILE()
	startupScript, err := cloud.RenderStartupScript(im.ctx, im.cluster, role)
	if err != nil {
		return nil, err
	}
	oneliners.FILE(startupScript)
	//imgID, err := strconv.Atoi(im.cluster.Spec.Cloud.InstanceImage)
	//if err != nil {
	//	return nil, errors.FromErr(err).WithContext(im.ctx).Err()
	//}
	oneliners.FILE()
	req := &godo.DropletCreateRequest{
		Name:   name,
		Region: im.cluster.Spec.Cloud.Zone,
		Size:   sku,
		//Image:  godo.DropletCreateImage{ID: imgID},
		Image: godo.DropletCreateImage{Slug: DROPLET_IMAGE_SLUG},
		SSHKeys: []godo.DropletCreateSSHKey{
			{Fingerprint: cloud.SSHKey(im.ctx).OpensshFingerprint},
			{Fingerprint: "0d:ff:0d:86:0c:f1:47:1d:85:67:1e:73:c6:0e:46:17"}, // tamal@beast
			{Fingerprint: "c0:19:c1:81:c5:2e:6d:d9:a6:db:3c:f5:c5:fd:c8:1d"}, // tamal@mbp
			{Fingerprint: "f6:66:c5:ad:e6:60:30:d9:ab:2c:7c:75:56:e2:d7:f3"}, // tamal@asus
			{Fingerprint: "80:b6:5a:c8:92:db:aa:fe:5f:d0:2e:99:95:de:ae:ab"}, // sanjid
			{Fingerprint: "93:e6:c6:95:5c:d1:ac:00:5e:23:8c:f7:d2:61:b7:07"}, // dipta
		},
		PrivateNetworking: true,
		IPv6:              false,
		UserData:          startupScript,
	}
	if _env.FromHost().IsPublic() {
		oneliners.FILE()
		req.SSHKeys = []godo.DropletCreateSSHKey{
			{Fingerprint: cloud.SSHKey(im.ctx).OpensshFingerprint},
		}
	}
	oneliners.FILE()
	droplet, resp, err := im.conn.client.Droplets.Create(gtx.TODO(), req)
	oneliners.FILE(droplet, resp, err)
	cloud.Logger(im.ctx).Debugln("do response", resp, " errors", err)
	cloud.Logger(im.ctx).Infof("Droplet %v created", droplet.Name)
	return droplet, err
}

func (im *instanceManager) applyTag(dropletID int) error {
	_, err := im.conn.client.Tags.TagResources(gtx.TODO(), "KubernetesCluster:"+im.cluster.Name, &godo.TagResourcesRequest{
		Resources: []godo.Resource{
			{
				ID:   strconv.Itoa(dropletID),
				Type: godo.DropletResourceType,
			},
		},
	})
	cloud.Logger(im.ctx).Infof("Tag %v applied to droplet %v", "KubernetesCluster:"+im.cluster.Name, dropletID)
	return err
}

func (im *instanceManager) assignReservedIP(ip string, dropletID int) error {
	action, resp, err := im.conn.client.FloatingIPActions.Assign(gtx.TODO(), ip, dropletID)
	if err != nil {
		return errors.FromErr(err).WithContext(im.ctx).Err()
	}
	cloud.Logger(im.ctx).Debugln("do response", resp, " errors", err)
	cloud.Logger(im.ctx).Debug("Created droplet with name", action.String())
	cloud.Logger(im.ctx).Infof("Reserved ip %v assigned to droplet %v", ip, dropletID)
	return nil
}

func (im *instanceManager) newKubeInstance(id int) (*api.Instance, error) {
	droplet, _, err := im.conn.client.Droplets.Get(gtx.TODO(), id)
	if err != nil {
		return nil, cloud.InstanceNotFound
	}
	return im.newKubeInstanceFromDroplet(droplet)
}

func (im *instanceManager) getInstanceId(name string) (int, error) {
	droplets, _, err := im.conn.client.Droplets.List(gtx.TODO(), &godo.ListOptions{})
	if err != nil {
		return -1, errors.FromErr(err).WithContext(im.ctx).Err()
	}

	for _, item := range droplets {
		fmt.Println(item.Name, name, "<><><><<><><><><><><><")
		if item.Name == name {
			return item.ID, nil
		}
	}
	return -1, errors.New("Instance not found").Err()
}

func (im *instanceManager) newKubeInstanceFromDroplet(droplet *godo.Droplet) (*api.Instance, error) {
	var externalIP, internalIP string
	externalIP, err := droplet.PublicIPv4()
	if err != nil {
		return nil, err
	}
	internalIP, err = droplet.PrivateIPv4()
	if err != nil {
		return nil, err
	}

	return &api.Instance{
		ObjectMeta: metav1.ObjectMeta{
			UID:  phid.NewKubeInstance(),
			Name: droplet.Name,
		},
		Spec: api.InstanceSpec{
			SKU: droplet.SizeSlug, // 512mb // convert to SKU
		},
		Status: api.InstanceStatus{
			ExternalID:    strconv.Itoa(droplet.ID),
			ExternalPhase: droplet.Status,
			PublicIP:      externalIP,
			PrivateIP:     internalIP,
			Phase:         api.InstancePhaseReady, // droplet.Status == active
		},
	}, nil
}

// reboot does not seem to run /etc/rc.local
func (im *instanceManager) reboot(id int) error {
	cloud.Logger(im.ctx).Infof("Rebooting instance %v", id)
	action, _, err := im.conn.client.DropletActions.Reboot(gtx.TODO(), id)
	if err != nil {
		return errors.FromErr(err).WithContext(im.ctx).Err()
	}
	cloud.Logger(im.ctx).Debugf("Instance status %v, %v", action, err)
	cloud.Logger(im.ctx).Infof("Instance %v reboot status %v", action.ResourceID, action.Status)
	return nil
}
