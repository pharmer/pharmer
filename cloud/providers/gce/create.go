package gce

import (
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/phid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (cm *ClusterManager) Create(req *proto.ClusterCreateRequest) error {
	var err error

	if cm.cluster, err = NewCluster(req); err != nil {
		return err
	}
	cm.namer = namer{cluster: cm.cluster}
	if cm.ctx, err = cloud.CreateCACertificates(cm.ctx, cm.cluster); err != nil {
		return err
	}
	if cm.ctx, err = cloud.CreateSSHKey(cm.ctx, cm.cluster); err != nil {
		return err
	}
	if _, err = cloud.Store(cm.ctx).Clusters().Create(cm.cluster); err != nil {
		return err
	}

	for _, ng := range req.NodeGroups {
		if ng.Count < 0 {
			ng.Count = 0
		}
		if ng.Count > maxInstancesPerMIG {
			ng.Count = maxInstancesPerMIG
		}
	}

	totalNodes := int64(0)
	for _, ng := range req.NodeGroups {
		totalNodes += ng.Count
		ig := api.InstanceGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:              ng.Sku + "-pool",
				UID:               phid.NewInstanceGroup(),
				CreationTimestamp: metav1.Time{Time: time.Now()},
				Labels: map[string]string{
					"node-role.kubernetes.io/node": "true",
				},
			},
			Spec: api.InstanceGroupSpec{
				SKU:           ng.Sku,
				Count:         ng.Count,
				SpotInstances: ng.SpotInstances,
				//DiskType:      "gp2",
				//DiskSize:      128,
			},
		}
		if _, err = cloud.Store(cm.ctx).InstanceGroups(req.Name).Create(&ig); err != nil {
			return err
		}
	}
	{
		ig := api.InstanceGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "master",
				UID:               phid.NewInstanceGroup(),
				CreationTimestamp: metav1.Time{Time: time.Now()},
				Labels: map[string]string{
					"node-role.kubernetes.io/master": "true",
				},
			},
			Spec: api.InstanceGroupSpec{
				Count:         1,
				SpotInstances: false,
				//DiskType:      "gp2",
				//DiskSize:      128,
			},
		}
		// check for instance count
		ig.Spec.SKU = "n1-standard-1"
		if totalNodes > 5 {
			ig.Spec.SKU = "n1-standard-2"
		}
		if totalNodes > 10 {
			ig.Spec.SKU = "n1-standard-4"
		}
		if totalNodes > 100 {
			ig.Spec.SKU = "n1-standard-8"
		}
		if totalNodes > 250 {
			ig.Spec.SKU = "n1-standard-16"
		}
		if totalNodes > 500 {
			ig.Spec.SKU = "n1-standard-32"
		}

		if _, err = cloud.Store(cm.ctx).InstanceGroups(req.Name).Create(&ig); err != nil {
			return err
		}
	}

	return nil
}

func (cm *ClusterManager) IsValid(cluster string) (bool, error) {
	return false, cloud.UnsupportedOperation
}
