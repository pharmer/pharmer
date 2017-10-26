package hetzner

import (
	"context"
	"fmt"
	"net"
	"strconv"

	hc "github.com/appscode/go-hetzner"
	_ssh "github.com/appscode/go/crypto/ssh"
	"github.com/appscode/go/errors"
	api "github.com/appscode/pharmer/apis/v1alpha1"
	. "github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/phid"
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
	servers, _, err := im.conn.client.Server.ListServers()
	if err != nil {
		return nil, err
	}
	for _, s := range servers {
		if !s.Cancelled && s.ServerIP == md.PublicIP {
			instance, err := im.newKubeInstanceFromSummary(s)
			if err != nil {
				return nil, err
			}
			if master {
				instance.Spec.Role = api.RoleMaster
			} else {
				instance.Spec.Role = api.RoleNode
			}
			return instance, nil

		}
	}
	return nil, errors.New("No instance found with name", md.Name).WithContext(im.ctx).Err()
}

func (im *instanceManager) createInstance(role, sku string) (*hc.Transaction, error) {
	tx, _, err := im.conn.client.Ordering.CreateTransaction(&hc.CreateTransactionRequest{
		ProductID:     sku,
		AuthorizedKey: []string{SSHKey(im.ctx).OpensshFingerprint},
		Dist:          im.cluster.Spec.Cloud.InstanceImage,
		Arch:          64,
		Lang:          "en",
		// Test:          true,
	})
	Logger(im.ctx).Infof("Instance with sku %v created", sku)
	return tx, err
}

func (im *instanceManager) storeConfigFile(serverIP, role string, signer ssh.Signer) error {
	Logger(im.ctx).Infof("Storing config file for server %v", serverIP)
	//cfg, err := im.cluster.StartupConfigResponse(role)
	//if err != nil {
	//	return errors.FromErr(err).WithContext(im.ctx).Err()
	//}
	//fmt.Println(">>>>>>>>>>>>>>>>>>>>>>>", cfg)
	cfg := ""

	file := fmt.Sprintf("/var/cache/kubernetes_context_%v_%v.yaml", im.cluster.Generation, role)
	stdOut, stdErr, code, err := _ssh.SCP(file, []byte(cfg), "root", serverIP+":22", signer)
	Logger(im.ctx).Debugf(stdOut, stdErr, code)
	return err
}

func (im *instanceManager) storeStartupScript(serverIP, sku, role string, signer ssh.Signer) error {
	Logger(im.ctx).Infof("Storing startup script for server %v", serverIP)
	startupScript := "" // TODO: fixit
	//startupScript, err := renderStartupScript(im.ctx, im.cluster, role)
	//if err != nil {
	//	return err
	//}
	fmt.Println(">>>>>>>>>>>>>>>>>>>>>>>", startupScript)

	file := "/var/cache/kubernetes_startupscript.sh"
	stdOut, stdErr, code, err := _ssh.SCP(file, []byte(startupScript), "root", serverIP+":22", signer)
	Logger(im.ctx).Debugf(stdOut, stdErr, code)
	return err
}

func (im *instanceManager) executeStartupScript(serverIP string, signer ssh.Signer) error {
	Logger(im.ctx).Infof("SSH execing start command %v", serverIP+":22")

	stdOut, stdErr, code, err := _ssh.Exec(`bash /var/cache/kubernetes_startupscript.sh`, "root", serverIP+":22", signer)
	Logger(im.ctx).Debugf(stdOut, stdErr, code)
	if err != nil {
		return errors.FromErr(err).WithContext(im.ctx).Err()
	}
	return nil
}

func (im *instanceManager) newKubeInstance(serverIP string) (*api.Node, error) {
	s, _, err := im.conn.client.Server.GetServer(serverIP)
	if err != nil {
		return nil, ErrNodeNotFound
	}
	return im.newKubeInstanceFromSummary(&s.ServerSummary)
}

func (im *instanceManager) newKubeInstanceFromSummary(droplet *hc.ServerSummary) (*api.Node, error) {
	return &api.Node{
		ObjectMeta: metav1.ObjectMeta{
			UID:  phid.NewKubeInstance(),
			Name: droplet.ServerName,
		},
		Spec: api.NodeSpec{
			SKU: droplet.Product,
		},
		Status: api.NodeStatus{
			ExternalID:    strconv.Itoa(droplet.ServerNumber),
			ExternalPhase: droplet.Status,
			PublicIP:      droplet.ServerIP,
			PrivateIP:     "",
			Phase:         api.NodeReady, // droplet.Status == active
		},
	}, nil
}
