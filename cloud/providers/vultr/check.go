package vultr

import (
	"fmt"

	api "github.com/appscode/pharmer/apis/v1alpha1"
	. "github.com/appscode/pharmer/cloud"
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

	return "", nil
}
