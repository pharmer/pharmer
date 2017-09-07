// Instance Provisioner: There is only 1 instance provisioner per cluster.Spec.
package scaleway

import (
	"context"
	"fmt"
	"net"

	sshtools "github.com/appscode/go/crypto/ssh"
	"github.com/appscode/go/errors"
	. "github.com/appscode/go/types"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/phid"
	"github.com/cenkalti/backoff"
	sapi "github.com/scaleway/scaleway-cli/pkg/api"
	"golang.org/x/crypto/ssh"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type instanceManager struct {
	ctx     context.Context
	cluster *api.Cluster
	conn    *cloudConnector
}

func (im *instanceManager) GetInstance(md *api.NodeStatus) (*api.Node, error) {
	master := net.ParseIP(md.Name) == nil

	var instance *api.Node
	backoff.Retry(func() (err error) {
		for {
			var servers *[]sapi.ScalewayServer
			servers, err = im.conn.client.GetServers(false, 0)
			if err != nil {
				return
			}
			for _, s := range *servers {
				if s.PrivateIP == md.PrivateIP {
					instance, err = im.newKubeInstanceFromServer(&s)
					if err != nil {
						return
					}
					if master {
						instance.Spec.Role = api.RoleMaster
					} else {
						instance.Spec.Role = api.RoleNode
					}
					return
				}
			}
		}
		return
	}, backoff.NewExponentialBackOff())

	if instance == nil {
		return nil, errors.New("No instance found with name", md.Name).Err()
	}
	return instance, nil
}

func (im *instanceManager) createInstance(name, role, sku string, ipid ...string) (string, error) {
	publicIPID := ""
	if len(ipid) > 0 {
		publicIPID = ipid[0]
	}
	serverID, err := im.conn.client.PostServer(sapi.ScalewayServerDefinition{
		Name:  name,
		Image: StringP(im.cluster.Spec.Cloud.InstanceImage),
		//Volumes map[string]string `json:"volumes,omitempty"`
		DynamicIPRequired: TrueP(),
		Bootscript:        StringP(im.conn.bootscriptID),
		Tags:              []string{"KubernetesCluster:" + im.cluster.Name},
		// Organization:   organization,
		CommercialType: sku,
		PublicIP:       publicIPID,
		//EnableIPV6 bool `json:"enable_ipv6,omitempty"`
		//SecurityGroup string `json:"security_group,omitempty"`
	})
	if err != nil {
		return "", errors.FromErr(err).WithContext(im.ctx).Err()
	}

	err = im.storeConfigFile(serverID, role)
	if err != nil {
		return "", errors.FromErr(err).WithContext(im.ctx).Err()
	}
	err = im.storeStartupScript(serverID, sku, role)
	if err != nil {
		return "", errors.FromErr(err).WithContext(im.ctx).Err()
	}
	err = im.conn.client.PostServerAction(serverID, "poweron")
	if err != nil {
		return "", errors.FromErr(err).WithContext(im.ctx).Err()
	}
	cloud.Logger(im.ctx).Infof("Instance %v created", name)
	return serverID, nil
}

func (im *instanceManager) storeConfigFile(serverID, role string) error {
	cloud.Logger(im.ctx).Infof("Storing config file for server %v", serverID)
	cfg := ""
	//cfg, err := im.cluster.StartupConfigResponse(role)
	//if err != nil {
	//	return errors.FromErr(err).WithContext(im.ctx).Err()
	//}
	dataKey := fmt.Sprintf("kubernetes_context_%v_%v.yaml", im.cluster.Generation, role)
	return im.conn.client.PatchUserdata(serverID, dataKey, []byte(cfg), false)
}

func (im *instanceManager) storeStartupScript(serverID, sku, role string) error {
	cloud.Logger(im.ctx).Infof("Storing startup script for server %v", serverID)
	startupScript := im.RenderStartupScript(sku, role)
	key := "kubernetes_startupscript.sh"
	return im.conn.client.PatchUserdata(serverID, key, []byte(startupScript), false)
}

func (im *instanceManager) RenderStartupScript(sku, role string) string {
	return "FixIT!"

	//	cmd := fmt.Sprintf(`CONFIG=$(/usr/bin/curl 169.254.42.42/user_data/kubernetes_context_%v_%v.yaml --local-port 1-1024)`, im.cluster.Generation, role)
	//	return fmt.Sprintf(`%v
	//systemctl start kube-installer.service
	//`, cloud.RenderKubeInstaller(im.cluster, sku, role, cmd))
}

func (im *instanceManager) executeStartupScript(instance *api.Node, signer ssh.Signer) error {
	cloud.Logger(im.ctx).Infof("SSH execing start command %v", instance.Status.PublicIP+":22")

	stdOut, stdErr, code, err := sshtools.Exec(`/usr/bin/curl 169.254.42.42/user_data/kubernetes_startupscript.sh --local-port 1-1024 2> /dev/null | bash`, "root", instance.Status.PublicIP+":22", signer)
	cloud.Logger(im.ctx).Infoln(stdOut, stdErr, code)
	if err != nil {
		return errors.FromErr(err).WithContext(im.ctx).Err()
	}
	return nil
}

func (im *instanceManager) newKubeInstance(id string) (*api.Node, error) {
	s, err := im.conn.client.GetServer(id)
	if err != nil {
		return nil, cloud.InstanceNotFound
	}
	return im.newKubeInstanceFromServer(s)
}

func (im *instanceManager) newKubeInstanceFromServer(droplet *sapi.ScalewayServer) (*api.Node, error) {
	return &api.Node{
		ObjectMeta: metav1.ObjectMeta{
			UID:  phid.NewKubeInstance(),
			Name: droplet.Name,
		},
		Spec: api.NodeSpec{
			SKU: droplet.CommercialType,
		},
		Status: api.NodeStatus{
			ExternalID:    droplet.Identifier,
			ExternalPhase: droplet.State,
			PublicIP:      droplet.PublicAddress.IP,
			PrivateIP:     droplet.PrivateIP,
			Phase:         api.NodeReady, // droplet.Status == active
		},
	}, nil
}
