package digitalocean

import (
	"fmt"

	"github.com/appscode/pharmer/api"
	. "github.com/appscode/pharmer/cloud"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	kc, err := cm.GetAdminClient()
	if err != nil {
		return "", err
	}

	masterInstance, err := kc.CoreV1().Nodes().Get(cm.namer.MasterName(), metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return cm.conn.ExecuteSSHCommand("kubeadm upgrade plan", masterInstance)
}
