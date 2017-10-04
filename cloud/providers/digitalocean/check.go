package digitalocean

import (
	"fmt"

	"github.com/appscode/pharmer/api"
	. "github.com/appscode/pharmer/cloud"
	"golang.org/x/crypto/ssh"
	apiv1 "k8s.io/api/core/v1"
)

func (cm *ClusterManager) Check(in *api.Cluster) (string, error) {
	var err error
	if in.Status.Phase == "" {
		return "", fmt.Errorf("cluster `%s` is in unknown phase", cm.cluster.Name)
	}
	if in.Status.Phase == api.ClusterDeleted {
		return "", nil
	}
	cm.cluster = in
	cm.namer = namer{cluster: cm.cluster}
	if cm.ctx, err = LoadCACertificates(cm.ctx, cm.cluster); err != nil {
		return "", err
	}
	if cm.ctx, err = LoadSSHKey(cm.ctx, cm.cluster); err != nil {
		return "", err
	}
	if cm.conn, err = NewConnector(cm.ctx, cm.cluster); err != nil {
		return "", err
	}

	resp, err := cm.checkClusterUpgrade()
	if err != nil {
		return "", err
	}
	//TODO: add other check

	return resp, nil
}

func (cm *ClusterManager) checkClusterUpgrade() (string, error) {
	keySigner, _ := ssh.ParsePrivateKey(SSHKey(cm.ctx).PrivateKey)
	config := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(keySigner),
		},
	}
	var externalIP string = ""
	for _, addr := range cm.cluster.Status.APIAddresses {
		if addr.Type == apiv1.NodeExternalIP {
			externalIP = addr.Address
			break
		}
	}
	return ExecuteCommand("kubeadm upgrade plan", fmt.Sprintf("%v:%v", externalIP, 22), config)
}
