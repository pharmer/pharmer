package framework

import (
	"time"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
)

var pool string

func (c *nodeGroupInvocaton) GetName() string {
	return "2gb-pool"
}

func (c *nodeGroupInvocaton) GetSkeleton() (*api.NodeGroup, error) {
	pool = c.GetName()
	ig := &api.NodeGroup{
		ObjectMeta: metav1.ObjectMeta{
			ClusterName:       c.ClusterName,
			UID:               uuid.NewUUID(),
			CreationTimestamp: metav1.Time{Time: time.Now()},
		},
		Spec: api.NodeGroupSpec{
			Nodes: int64(1),
			Template: api.NodeTemplateSpec{
				Spec: api.NodeSpec{
					SKU: "2gb",
				},
			},
		},
	}
	ig.ObjectMeta.Name = pool
	ig.ObjectMeta.Labels = map[string]string{
		api.RoleNodeKey: "",
	}
	return ig, nil
}

func (c *nodeGroupInvocaton) Update(ng *api.NodeGroup) error {
	ng.Spec.Nodes = int64(2)
	_, err := c.Storage.NodeGroups(c.clusterName).Update(ng)
	return err
}

func (c *nodeGroupInvocaton) CheckUpdate(ng *api.NodeGroup) error {
	if ng.Spec.Nodes == int64(2) {
		return nil
	}
	return errors.Errorf("node group was not updated")
}

func (c *nodeGroupInvocaton) UpdateStatus(ng *api.NodeGroup) error {
	ng.Status = api.NodeGroupStatus{
		Nodes: int64(2),
	}
	_, err := c.Storage.NodeGroups(c.clusterName).UpdateStatus(ng)
	return err
}

func (c *nodeGroupInvocaton) CheckUpdateStatus(ng *api.NodeGroup) error {
	if ng.Status.Nodes == int64(2) {
		return nil
	}
	return errors.Errorf("node group status was not updated")
}

func (c *nodeGroupInvocaton) List() error {
	ngs, err := c.Storage.NodeGroups(c.clusterName).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	if len(ngs) < 1 {
		return errors.Errorf("can't list node groups")
	}
	return nil
}

func (c *nodeGroupInvocaton) Create(ng *api.NodeGroup) error {
	_, err := c.Storage.NodeGroups(c.ClusterName).Create(ng)
	return err
}
