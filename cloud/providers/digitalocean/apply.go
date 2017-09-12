package digitalocean

import (
	gtx "context"
	"fmt"
	"os"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/pharmer/api"
	. "github.com/appscode/pharmer/cloud"
	"github.com/digitalocean/godo"
	"github.com/tamalsaha/go-oneliners"
)

func (cm *ClusterManager) Apply(in *api.Cluster, dryRun bool) error {
	_, err := cm.apply(in, api.DryRun)
	return err
}

func (cm *ClusterManager) apply(in *api.Cluster, rt api.RunType) (acts []api.Action, err error) {
	cm.cluster = in
	if cm.conn, err = NewConnector(cm.ctx, cm.cluster); err != nil {
		return
	}

	defer func(releaseReservedIp bool) {
		if cm.cluster.Status.Phase == api.ClusterPending {
			cm.cluster.Status.Phase = api.ClusterFailing
		}
		Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
		Logger(cm.ctx).Infof("Cluster %v is %v", cm.cluster.Name, cm.cluster.Status.Phase)
		if cm.cluster.Status.Phase != api.ClusterReady {
			Logger(cm.ctx).Infof("Cluster %v is deleting", cm.cluster.Name)
			cm.Delete(&proto.ClusterDeleteRequest{
				Name:              cm.cluster.Name,
				ReleaseReservedIp: releaseReservedIp,
			})
		}
	}(cm.cluster.Spec.MasterReservedIP == "auto")

	//cm.cluster.Spec.Cloud.InstanceImage, err = cm.conn.getInstanceImage()
	//if err != nil {
	//	cm.cluster.Status.Reason = err.Error()
	//	return
	//}
	//Logger(cm.ctx).Infof("Image %v is using to create instance", cm.cluster.Spec.Cloud.InstanceImage)

	err = cm.importPublicKey()
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return
	}

	// ignore errors, since tags are simply informational.
	cm.createTags()

	err = cm.reserveIP()
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return
	}

	// -------------------------------------------------------------------ASSETS
	im := &instanceManager{ctx: cm.ctx, cluster: cm.cluster, conn: cm.conn, namer: cm.namer}

	Logger(cm.ctx).Info("Creating master instance")
	masterDroplet, err := im.createInstance(cm.cluster.Spec.KubernetesMasterName, api.RoleMaster, cm.cluster.Spec.MasterSKU)
	if err != nil {
		oneliners.FILE(err)
		cm.cluster.Status.Reason = err.Error()
		return
	}
	oneliners.FILE()
	if err = cm.conn.waitForInstance(masterDroplet.ID, "active"); err != nil {
		oneliners.FILE(err)
		cm.cluster.Status.Reason = err.Error()
		return
	}
	oneliners.FILE()
	im.applyTag(masterDroplet.ID)
	oneliners.FILE()
	if cm.cluster.Spec.MasterReservedIP != "" {
		oneliners.FILE()
		if err = im.assignReservedIP(cm.cluster.Spec.MasterReservedIP, masterDroplet.ID); err != nil {
			oneliners.FILE(err)
			cm.cluster.Status.Reason = err.Error()
			return
		}
	}
	masterInstance, err := im.newKubeInstance(masterDroplet.ID)
	if err != nil {
		oneliners.FILE(err)
		cm.cluster.Status.Reason = err.Error()
		return
	}
	masterInstance.Spec.Role = api.RoleMaster
	cm.cluster.Spec.MasterExternalIP = masterInstance.Status.PublicIP
	cm.cluster.Spec.MasterInternalIP = masterInstance.Status.PrivateIP
	fmt.Println("Master EXTERNAL IP ================", cm.cluster.Spec.MasterExternalIP, "<><><><>", cm.cluster.Spec.MasterReservedIP)
	Store(cm.ctx).Instances(cm.cluster.Name).Create(masterInstance)

	//if err = GenClusterCerts(cm.ctx, cm.cluster); err != nil {
	//	cm.cluster.Status.Reason = err.Error()
	//	return
	//}
	err = EnsureARecord(cm.ctx, cm.cluster, masterInstance) // works for reserved or non-reserved mode
	if err != nil {
		oneliners.FILE(err)
		cm.cluster.Status.Reason = err.Error()
		return
	}
	oneliners.FILE()
	// needed to get master_internal_ip
	if _, err = Store(cm.ctx).Clusters().UpdateStatus(cm.cluster); err != nil {
		oneliners.FILE(err)
		cm.cluster.Status.Reason = err.Error()
		return
	}
	//cm.UploadStartupConfig(cm.cluster)
	//
	//// reboot master to use cert with internal_ip as SANS
	//time.Sleep(60 * time.Second)

	//Logger(cm.ctx).Info("Rebooting master instance")
	//if err = im.reboot(masterDroplet.ID); err != nil {
	//	cm.cluster.Status.Reason = err.Error()
	//	return
	//}
	//Logger(cm.ctx).Info("Rebooted master instance")

	// start nodes
	//for _, ng := range req.NodeGroups {
	//	Logger(cm.ctx).Infof("Creating %v node with sku %v", ng.Count, ng.Sku)
	//	igm := &NodeGroupManager{
	//		cm: cm,
	//		instance: Instance{
	//			Type: InstanceType{
	//				ContextVersion: cm.cluster.Generation,
	//				Sku:            ng.Sku,
	//
	//				Master:       false,
	//				SpotInstance: false,
	//			},
	//			Stats: GroupStats{
	//				Count: ng.Count,
	//			},
	//		},
	//		im: im,
	//	}
	//	err = igm.AdjustNodeGroup()
	//}

	Logger(cm.ctx).Info("Waiting for cluster initialization")

	// Wait for master A record to propagate
	if err = EnsureDnsIPLookup(cm.ctx, cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return
	}

	os.Exit(1)

	// wait for nodes to start
	if err = WaitForReadyMaster(cm.ctx, cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return
	}
	cm.cluster.Status.Phase = api.ClusterReady
	return
}

func (cm *ClusterManager) importPublicKey() error {
	key, resp, err := cm.conn.client.Keys.Create(gtx.TODO(), &godo.KeyCreateRequest{
		Name:      cm.cluster.Status.SSHKeyExternalID,
		PublicKey: string(SSHKey(cm.ctx).PublicKey),
	})
	if err != nil {
		return err
	}
	Logger(cm.ctx).Debugln("DO response", resp, " errors", err)
	Logger(cm.ctx).Debugf("Created new ssh key with name=%v and id=%v", cm.cluster.Status.SSHKeyExternalID, key.ID)
	Logger(cm.ctx).Info("SSH public key added")
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
			return err
		}
		Logger(cm.ctx).Infof("Tag %v created", tag)
	}
	return nil
}

func (cm *ClusterManager) reserveIP() error {
	if cm.cluster.Spec.MasterReservedIP == "auto" {
		fip, resp, err := cm.conn.client.FloatingIPs.Create(gtx.TODO(), &godo.FloatingIPCreateRequest{
			Region: cm.cluster.Spec.Cloud.Region,
		})
		if err != nil {
			return err
		}
		Logger(cm.ctx).Debugln("DO response", resp, " errors", err)
		Logger(cm.ctx).Infof("New floating ip %v reserved", fip.IP)
		cm.cluster.Spec.MasterReservedIP = fip.IP
	}
	return nil
}
