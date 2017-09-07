package vultr

import (
	"context"
	"net"
	"strconv"

	gv "github.com/JamesClonk/vultr/lib"
	_env "github.com/appscode/go/env"
	"github.com/appscode/go/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/phid"
	"github.com/cenkalti/backoff"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type instanceManager struct {
	ctx     context.Context
	cluster *api.Cluster
	conn    *cloudConnector
	namer   namer
}

func (im *instanceManager) GetInstance(md *api.NodeStatus) (*api.Node, error) {
	master := net.ParseIP(md.Name) == nil

	var instance *api.Node
	backoff.Retry(func() (err error) {
		servers, err := im.conn.client.GetServers()
		if err != nil {
			return
		}
		for _, server := range servers {
			if server.InternalIP == md.PrivateIP {
				instance, err = im.newKubeInstance(&server)
				if master {
					instance.Spec.Role = api.RoleMaster
				} else {
					instance.Spec.Role = api.RoleNode
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
	cloud.Logger(im.ctx).Infof("creating StackScript for sku %v role %v", sku, role)
	script, err := RenderStartupScript(im.ctx, im.cluster, role)
	if err != nil {
		return 0, err
	}
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

func (im *instanceManager) createInstance(name, sku string, scriptID int) (string, error) {
	regionID, err := strconv.Atoi(im.cluster.Spec.Cloud.Zone)
	if err != nil {
		return "", errors.FromErr(err).WithContext(im.ctx).Err()
	}
	planID, err := strconv.Atoi(sku)
	if err != nil {
		return "", errors.FromErr(err).WithContext(im.ctx).Err()
	}
	osID, err := strconv.Atoi(im.cluster.Spec.Cloud.InstanceImage)
	if err != nil {
		return "", errors.FromErr(err).WithContext(im.ctx).Err()
	}
	opts := &gv.ServerOptions{
		SSHKey:               im.cluster.Status.SSHKeyExternalID + ",57dcbce7cd3b6,58027d56a1190,58a498ec7ee19",
		PrivateNetworking:    true,
		DontNotifyOnActivate: false,
		Script:               scriptID,
		Hostname:             name,
		Tag:                  im.cluster.Name,
	}
	if _env.FromHost().IsPublic() {
		opts.SSHKey = im.cluster.Status.SSHKeyExternalID
	}
	resp, err := im.conn.client.CreateServer(
		name,
		regionID,
		planID,
		osID,
		opts)
	cloud.Logger(im.ctx).Debugln("do response", resp, " errors", err)
	cloud.Logger(im.ctx).Debug("Created droplet with name", resp.ID)
	cloud.Logger(im.ctx).Infof("DO droplet %v created", name)
	return resp.ID, err
}

func (im *instanceManager) assignReservedIP(ip, serverId string) error {
	err := im.conn.client.AttachReservedIP(ip, serverId)
	if err != nil {
		return errors.FromErr(err).WithContext(im.ctx).Err()
	}
	cloud.Logger(im.ctx).Infof("Reserved ip %v assigned to %v", ip, serverId)
	return nil
}

func (im *instanceManager) newKubeInstance(server *gv.Server) (*api.Node, error) {
	return &api.Node{
		ObjectMeta: metav1.ObjectMeta{
			UID:  phid.NewKubeInstance(),
			Name: server.Name,
		},
		Spec: api.NodeSpec{
			SKU: strconv.Itoa(server.PlanID), // 512mb // convert to SKU
		},
		Status: api.NodeStatus{
			ExternalID:    server.ID,
			ExternalPhase: server.Status + "|" + server.PowerStatus,
			PublicIP:      server.MainIP,
			PrivateIP:     server.InternalIP,
			Phase:         api.NodeReady, // active
		},
	}, nil
}

// reboot does not seem to run /etc/rc.local
func (im *instanceManager) reboot(id string) error {
	cloud.Logger(im.ctx).Infof("Rebooting instance %v", id)
	err := im.conn.client.RebootServer(id)
	if err != nil {
		return errors.FromErr(err).WithContext(im.ctx).Err()
	}
	return nil
}
