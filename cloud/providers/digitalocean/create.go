package digitalocean

import (
	gtx "context"
	"fmt"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/go/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/phid"
	"github.com/digitalocean/godo"
	"github.com/tamalsaha/go-oneliners"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (cm *ClusterManager) Create(req *proto.ClusterCreateRequest) error {
	cm.cluster = &api.Cluster{
		ObjectMeta: api.ObjectMeta{
			Name:              req.Name,
			UID:               phid.NewKubeCluster(),
			CreationTimestamp: metav1.Time{Time: time.Now()},
		},
		Spec: api.ClusterSpec{
			CredentialName: req.CredentialUid,
		},
	}
	cm.cluster.Spec.Zone = req.Zone
	api.AssignTypeKind(cm.cluster)
	if _, err := cloud.Store(cm.ctx).Clusters().Create(cm.cluster); err != nil {
		oneliners.FILE(err)
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	err := cm.initContext(req)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ins = make([]*api.Instance, 0)
	cm.conn, err = NewConnector(cm.ctx, cm.cluster)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)

	defer func(releaseReservedIp bool) {
		if cm.cluster.Status.Phase == api.ClusterPhasePending {
			cm.cluster.Status.Phase = api.ClusterPhaseFailing
		}
		cloud.Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
		cloud.Store(cm.ctx).Instances(cm.cluster.Name).SaveInstances(cm.ins)
		cloud.Logger(cm.ctx).Infof("Cluster %v is %v", cm.cluster.Name, cm.cluster.Status.Phase)
		if cm.cluster.Status.Phase != api.ClusterPhaseReady {
			cloud.Logger(cm.ctx).Infof("Cluster %v is deleting", cm.cluster.Name)
			cm.Delete(&proto.ClusterDeleteRequest{
				Name:              cm.cluster.Name,
				ReleaseReservedIp: releaseReservedIp,
			})
		}
	}(cm.cluster.Spec.MasterReservedIP == "auto")

	//cm.cluster.Spec.InstanceImage, err = cm.conn.getInstanceImage()
	//if err != nil {
	//	cm.cluster.Status.Reason = err.Error()
	//	return errors.FromErr(err).WithContext(cm.ctx).Err()
	//}
	//cloud.Logger(cm.ctx).Infof("Image %v is using to create instance", cm.cluster.Spec.InstanceImage)

	err = cm.importPublicKey()
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// ignore errors, since tags are simply informational.
	cm.createTags()

	err = cm.reserveIP()
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// ------------------------Generate Certs and upload in Firebase @dipta----------------------

	if cm.ctx, err = cloud.GenClusterCerts(cm.ctx, cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// -------------------------------------------------------------------ASSETS
	im := &instanceManager{ctx: cm.ctx, cluster: cm.cluster, conn: cm.conn, namer: cm.namer}

	cloud.Logger(cm.ctx).Info("Creating master instance")
	masterDroplet, err := im.createInstance(cm.cluster.Spec.KubernetesMasterName, api.RoleKubernetesMaster, cm.cluster.Spec.MasterSKU)
	if err != nil {
		oneliners.FILE(err)
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	oneliners.FILE()
	if err = cm.conn.waitForInstance(masterDroplet.ID, "active"); err != nil {
		oneliners.FILE(err)
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	oneliners.FILE()
	im.applyTag(masterDroplet.ID)
	oneliners.FILE()
	if cm.cluster.Spec.MasterReservedIP != "" {
		oneliners.FILE()
		if err = im.assignReservedIP(cm.cluster.Spec.MasterReservedIP, masterDroplet.ID); err != nil {
			oneliners.FILE(err)
			cm.cluster.Status.Reason = err.Error()
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}
	masterInstance, err := im.newKubeInstance(masterDroplet.ID)
	if err != nil {
		oneliners.FILE(err)
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterInstance.Spec.Role = api.RoleKubernetesMaster
	cm.cluster.Spec.MasterExternalIP = masterInstance.Status.ExternalIP
	cm.cluster.Spec.MasterInternalIP = masterInstance.Status.InternalIP
	fmt.Println("Master EXTERNAL IP ================", cm.cluster.Spec.MasterExternalIP, "<><><><>", cm.cluster.Spec.MasterReservedIP)
	cm.ins = append(cm.ins, masterInstance)

	//if err = cloud.GenClusterCerts(cm.ctx, cm.cluster); err != nil {
	//	cm.cluster.Status.Reason = err.Error()
	//	return errors.FromErr(err).WithContext(cm.ctx).Err()
	//}
	err = cloud.EnsureARecord(cm.ctx, cm.cluster, masterInstance) // works for reserved or non-reserved mode
	if err != nil {
		oneliners.FILE(err)
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	oneliners.FILE()
	// needed to get master_internal_ip
	if _, err = cloud.Store(cm.ctx).Clusters().UpdateStatus(cm.cluster); err != nil {
		oneliners.FILE(err)
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	//cm.UploadStartupConfig(cm.cluster)
	//
	//// reboot master to use cert with internal_ip as SANS
	//time.Sleep(60 * time.Second)

	//cloud.Logger(cm.ctx).Info("Rebooting master instance")
	//if err = im.reboot(masterDroplet.ID); err != nil {
	//	cm.cluster.Status.Reason = err.Error()
	//	return errors.FromErr(err).WithContext(cm.ctx).Err()
	//}
	//cloud.Logger(cm.ctx).Info("Rebooted master instance")
	cm.ins = append(cm.ins, masterInstance)
	// start nodes
	for _, ng := range req.NodeGroups {
		cloud.Logger(cm.ctx).Infof("Creating %v node with sku %v", ng.Count, ng.Sku)
		igm := &InstanceGroupManager{
			cm: cm,
			instance: cloud.Instance{
				Type: cloud.InstanceType{
					ContextVersion: cm.cluster.Spec.ResourceVersion,
					Sku:            ng.Sku,

					Master:       false,
					SpotInstance: false,
				},
				Stats: cloud.GroupStats{
					Count: ng.Count,
				},
			},
			im: im,
		}
		err = igm.AdjustInstanceGroup()
	}

	cloud.Logger(cm.ctx).Info("Waiting for cluster initialization")

	// Wait for master A record to propagate
	if err := cloud.EnsureDnsIPLookup(cm.ctx, cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// wait for nodes to start
	if err := cloud.WaitForReadyMaster(cm.ctx, cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.cluster.Status.Phase = api.ClusterPhaseReady
	return nil
}

func (cm *ClusterManager) importPublicKey() error {
	key, resp, err := cm.conn.client.Keys.Create(gtx.TODO(), &godo.KeyCreateRequest{
		Name:      cm.cluster.Spec.SSHKeyExternalID,
		PublicKey: string(cm.cluster.Spec.SSHKey.PublicKey),
	})
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Debugln("DO response", resp, " errors", err)
	cloud.Logger(cm.ctx).Debugf("Created new ssh key with name=%v and id=%v", cm.cluster.Spec.SSHKeyExternalID, key.ID)
	cloud.Logger(cm.ctx).Info("SSH public key added")
	return nil
}

func (cm *ClusterManager) createTags() error {
	tag := "KubernetesCluster:" + cm.cluster.Name
	_, _, err := cm.conn.client.Tags.Get(gtx.TODO(), tag)
	if err != nil {
		// Tag does not already exist
		_, _, err := cm.conn.client.Tags.Create(gtx.TODO(), &godo.TagCreateRequest{
			Name: tag,
		})
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cloud.Logger(cm.ctx).Infof("Tag %v created", tag)
	}
	return nil
}

func (cm *ClusterManager) reserveIP() error {
	if cm.cluster.Spec.MasterReservedIP == "auto" {
		fip, resp, err := cm.conn.client.FloatingIPs.Create(gtx.TODO(), &godo.FloatingIPCreateRequest{
			Region: cm.cluster.Spec.Region,
		})
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cloud.Logger(cm.ctx).Debugln("DO response", resp, " errors", err)
		cloud.Logger(cm.ctx).Infof("New floating ip %v reserved", fip.IP)
		cm.cluster.Spec.MasterReservedIP = fip.IP
	}
	return nil
}
