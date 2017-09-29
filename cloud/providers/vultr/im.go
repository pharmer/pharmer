package vultr

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	gv "github.com/JamesClonk/vultr/lib"
	_env "github.com/appscode/go/env"
	"github.com/appscode/go/errors"
	api "github.com/appscode/pharmer/apis/v1alpha1"
	. "github.com/appscode/pharmer/cloud"
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

func (im *instanceManager) getStartupScript(sku, role string) (bool, int, error) {
	scripts, err := im.conn.client.GetStartupScripts()
	if err != nil {
		return false, -1, err
	}
	for _, script := range scripts {
		if script.Name == im.namer.StartupScriptName(sku, role) {
			scriptID, _ := strconv.Atoi(script.ID)
			return true, scriptID, nil
		}
	}
	return false, -1, nil
}

func (im *instanceManager) createStartupScript(sku, role string) (int, error) {
	instanceGroup := ""
	if role == api.RoleNode {
		instanceGroup = im.namer.GetNodeGroupName(sku)
	}
	Logger(im.ctx).Infof("creating StackScript for sku %v role %v", sku, role)
	//script, err := renderStartupScript(im.ctx, im.cluster, role)
	//if err != nil {
	//	return 0, err
	//}
	// TODO: Fix it: conn.renderStartupScript()
	script := ""
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

func (im *instanceManager) getInstance(name string) (string, error) {
	servers, err := im.conn.client.GetServersByTag(im.cluster.Name)
	fmt.Println(err, servers)
	if err != nil {
		return "", err
	}
	for _, server := range servers {
		fmt.Println(server.Name, "***")
		if server.Name == name {
			return server.ID, nil
		}
	}
	return "", nil
}

func (im *instanceManager) getPublicKey() (bool, string, error) {
	keys, err := im.conn.client.GetSSHKeys()
	if err != nil {
		return false, "", err
	}
	for _, key := range keys {
		if key.Name == im.cluster.Status.SSHKeyExternalID {
			return true, key.ID, nil
		}
	}
	return false, "", nil
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

	_, sshKeyID, err := im.getPublicKey()
	fmt.Println(sshKeyID, "*************")
	if err != nil {
		return "", errors.FromErr(err).WithContext(im.ctx).Err()
	}

	opts := &gv.ServerOptions{
		SSHKey:               sshKeyID + ",57dcbce7cd3b6,58027d56a1190,58a498ec7ee19,57ee2df762851",
		PrivateNetworking:    true,
		DontNotifyOnActivate: false,
		Script:               scriptID,
		Hostname:             name,
		Tag:                  im.cluster.Name,
	}
	if _env.FromHost().IsPublic() {
		opts.SSHKey = sshKeyID
	}
	resp, err := im.conn.client.CreateServer(
		name,
		regionID,
		planID,
		osID,
		opts)
	Logger(im.ctx).Debugln("Vultr response", resp, " errors", err)
	Logger(im.ctx).Debug("Created server with name", resp.ID)
	Logger(im.ctx).Infof("Vultr server %v created", name)
	return resp.ID, err
}

func (im *instanceManager) assignReservedIP(ip, serverId string) error {
	err := im.conn.client.AttachReservedIP(ip, serverId)
	if err != nil {
		return errors.FromErr(err).WithContext(im.ctx).Err()
	}
	Logger(im.ctx).Infof("Reserved ip %v assigned to %v", ip, serverId)
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

func (im *instanceManager) GetNodeGroup(instanceGroup string) (bool, map[string]*api.Node, error) {
	var flag bool = false
	existingNGs := make(map[string]*api.Node)
	servers, err := im.conn.client.GetServersByTag(im.cluster.Name)
	if err != nil {
		return flag, existingNGs, errors.FromErr(err).WithContext(im.ctx).Err()
	}

	for _, item := range servers {
		if strings.HasPrefix(item.Name, instanceGroup) {
			flag = true
			instance, err := im.newKubeInstance(&item)
			if err != nil {
				return flag, existingNGs, errors.FromErr(err).WithContext(im.ctx).Err()
			}
			instance.Spec.Role = api.RoleNode
			existingNGs[item.Name] = instance
		}

	}
	return flag, existingNGs, nil
}

func (im *instanceManager) deleteServer(id string) error {
	return backoff.Retry(func() error {
		err := im.conn.client.DeleteServer(id)
		if err != nil {
			return err
		}

		return nil
	}, backoff.NewExponentialBackOff())
}

// reboot does not seem to run /etc/rc.local
func (im *instanceManager) reboot(id string) error {
	return nil
	Logger(im.ctx).Infof("Rebooting instance %v", id)
	err := im.conn.client.RebootServer(id)
	if err != nil {
		return errors.FromErr(err).WithContext(im.ctx).Err()
	}
	return nil
}
