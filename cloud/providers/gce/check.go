package gce

import (
	"fmt"

	"github.com/appscode/pharmer/api"
	. "github.com/appscode/pharmer/cloud"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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
	nodeGroups, err := Store(cm.ctx).NodeGroups(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return "", err
	}
	masterNG := FindMasterNodeGroup(nodeGroups)

	nodes, err := kc.CoreV1().Nodes().List(metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			api.NodeLabelKey_NodeGroup: masterNG.Name,
			//api.RoleMasterKey:          "",
		}).String(),
	})
	if err != nil {
		return "", err
	}
	if len(nodes.Items) == 0 {
		return "", fmt.Errorf("Master node not found")
	}
	return cm.conn.ExecuteSSHCommand("kubeadm upgrade plan", &nodes.Items[0])
}
