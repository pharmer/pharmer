package digitalocean

import (
	gtx "context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/appscode/pharmer/api"
	. "github.com/appscode/pharmer/cloud"
	"github.com/cenkalti/backoff"
	"github.com/digitalocean/godo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (cm *ClusterManager) Apply(in *api.Cluster, dryRun bool) error {
	var err error
	if dryRun {
		action, err := cm.apply(in, api.DryRun)
		fmt.Println(err)
		jm, err := json.Marshal(action)
		fmt.Println(string(jm), err)
	} else {
		_, err = cm.apply(in, api.StdRun)
	}
	return err
}

func (cm *ClusterManager) apply(in *api.Cluster, rt api.RunType) (acts []api.Action, err error) {
	var (
		clusterDelete = false
	)
	if in.DeletionTimestamp != nil && in.Status.Phase != api.ClusterDeleted {
		clusterDelete = true
	}

	cm.cluster = in
	cm.namer = namer{cluster: cm.cluster}
	if cm.ctx, err = LoadCACertificates(cm.ctx, cm.cluster); err != nil {
		return
	}
	if cm.ctx, err = LoadSSHKey(cm.ctx, cm.cluster); err != nil {
		return
	}
	if cm.conn, err = NewConnector(cm.ctx, cm.cluster); err != nil {
		return
	}
	acts = make([]api.Action, 0)

	if cm.cluster.Status.Phase == "" {
		err = fmt.Errorf("cluster `%s` is in unknown status", cm.cluster.Name)
		return
	}

	if cm.cluster.Status.Phase == api.ClusterPending {
		// create cluster
	}

	/*defer func(releaseReservedIp bool) {
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
	}(cm.cluster.Spec.MasterReservedIP == "auto")*/

	//cm.cluster.Spec.Cloud.InstanceImage, err = cm.conn.getInstanceImage()
	//if err != nil {
	//	cm.cluster.Status.Reason = err.Error()
	//	return
	//}
	//Logger(cm.ctx).Infof("Image %v is using to create instance", cm.cluster.Spec.Cloud.InstanceImage)

	if found, _ := cm.getPublicKey(); !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Public key",
			Message:  "Public key will be imported",
		})
		if rt != api.DryRun {
			err = cm.importPublicKey()
			if err != nil {
				cm.cluster.Status.Reason = err.Error()
				return
			}
		}
	} else {
		if clusterDelete {
			acts = append(acts, api.Action{
				Action:   api.ActionDelete,
				Resource: "Public key",
				Message:  "Public key will be deleted",
			})
		} else {
			acts = append(acts, api.Action{
				Action:   api.ActionNOP,
				Resource: "Public key",
				Message:  "Public key found",
			})
		}
	}

	// ignore errors, since tags are simply informational.
	if found, _ := cm.getTags(); !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Tag",
			Message:  fmt.Sprintf("Tag %v will be added", "KubernetesCluster:"+cm.cluster.Name),
		})
		if rt != api.DryRun {
			cm.createTags()
		}
	} else {
		if clusterDelete {
			acts = append(acts, api.Action{
				Action:   api.ActionDelete,
				Resource: "Tag",
				Message:  fmt.Sprintf("Tag %v will be deleted", "KubernetesCluster:"+cm.cluster.Name),
			})
		} else {
			acts = append(acts, api.Action{
				Action:   api.ActionNOP,
				Resource: "Tag",
				Message:  fmt.Sprintf("Tag %v found", "KubernetesCluster:"+cm.cluster.Name),
			})
		}
	}

	if found, _ := cm.getReserveIP(); !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Reserve IP",
			Message:  fmt.Sprintf("Not found, MasterReservedIP = ", cm.cluster.Spec.MasterReservedIP),
		})
		if rt != api.DryRun {
			err = cm.reserveIP()
			if err != nil {
				cm.cluster.Status.Reason = err.Error()
				return
			}
		}
	} else {
		if clusterDelete {
			acts = append(acts, api.Action{
				Action:   api.ActionDelete,
				Resource: "Reserve IP",
				Message:  fmt.Sprintf(" MasterReservedIP %v will be deleted ", cm.cluster.Spec.MasterReservedIP),
			})
		} else {
			acts = append(acts, api.Action{
				Action:   api.ActionNOP,
				Resource: "Reserve IP",
				Message:  fmt.Sprintf("Found, MasterReservedIP = ", cm.cluster.Spec.MasterReservedIP),
			})
		}
	}

	// -------------------------------------------------------------------ASSETS
	im := &instanceManager{ctx: cm.ctx, cluster: cm.cluster, conn: cm.conn, namer: cm.namer}

	nodeGroups, err := Store(cm.ctx).NodeGroups(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return
	}

	var masterNG *api.NodeGroup
	var totalNodes int64 = 0
	for _, ng := range nodeGroups {
		if ng.IsMaster() {
			masterNG = ng
		} else {
			totalNodes += ng.Spec.Nodes
		}
	}
	fmt.Println(totalNodes)

	cm.cluster.Spec.MasterSKU = "2gb"

	if id, err := im.getInstanceId(cm.cluster.Spec.KubernetesMasterName); err != nil {
		Logger(cm.ctx).Info("Creating master instance")
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Master Instance",
			Message:  fmt.Sprintf("Master instance with name %v will be created", cm.cluster.Spec.KubernetesMasterName),
		})
		if rt != api.DryRun {
			masterDroplet, err := im.createInstance(cm.cluster.Spec.KubernetesMasterName, api.RoleMaster, cm.cluster.Spec.MasterSKU)
			if err != nil {
				cm.cluster.Status.Reason = err.Error()
				return acts, err
			}

			if err = cm.conn.waitForInstance(masterDroplet.ID, "active"); err != nil {
				cm.cluster.Status.Reason = err.Error()
				return acts, err
			}
			im.applyTag(masterDroplet.ID)
			if cm.cluster.Spec.MasterReservedIP != "" {
				if err = im.assignReservedIP(cm.cluster.Spec.MasterReservedIP, masterDroplet.ID); err != nil {
					cm.cluster.Status.Reason = err.Error()
					return acts, err
				}
			}
			masterInstance, err := im.newKubeInstance(masterDroplet.ID)
			if err != nil {
				cm.cluster.Status.Reason = err.Error()
				return acts, err
			}
			masterInstance.Spec.Role = api.RoleMaster
			cm.cluster.Spec.MasterExternalIP = masterInstance.Status.PublicIP
			cm.cluster.Spec.MasterInternalIP = masterInstance.Status.PrivateIP
			fmt.Println("Master EXTERNAL IP ================", cm.cluster.Spec.MasterExternalIP, "<><><><>", cm.cluster.Spec.MasterReservedIP)
			Store(cm.ctx).Instances(cm.cluster.Name).Create(masterInstance)

			err = EnsureARecord(cm.ctx, cm.cluster, masterInstance) // works for reserved or non-reserved mode
			if err != nil {
				cm.cluster.Status.Reason = err.Error()
				return acts, err
			}

			// Wait for master A record to propagate
			if err = EnsureDnsIPLookup(cm.ctx, cm.cluster); err != nil {
				cm.cluster.Status.Reason = err.Error()
				return acts, err
			}

			// wait for nodes to start
			if err = WaitForReadyMaster(cm.ctx, cm.cluster); err != nil {
				cm.cluster.Status.Reason = err.Error()
				return acts, err
			}
			masterNG.Status.Nodes = int32(1)
			Store(cm.ctx).NodeGroups(cm.cluster.Name).Update(masterNG)
			Store(cm.ctx).Instances(cm.cluster.Name).Create(masterInstance)
			// needed to get master_internal_ip
			if _, err = Store(cm.ctx).Clusters().UpdateStatus(cm.cluster); err != nil {
				cm.cluster.Status.Reason = err.Error()
				return acts, err
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Master Instance",
			Message:  fmt.Sprintf("Found master instance with name %v", cm.cluster.Spec.KubernetesMasterName),
		})
		masterInstance, _ := im.newKubeInstance(id)
		masterInstance.Spec.Role = api.RoleMaster
		cm.cluster.Spec.MasterExternalIP = masterInstance.Status.PublicIP
		cm.cluster.Spec.MasterInternalIP = masterInstance.Status.PrivateIP

		if clusterDelete {
			acts = append(acts, api.Action{
				Action:   api.ActionDelete,
				Resource: "Master Instance",
				Message:  fmt.Sprintf("Will delete master instance with name %v", cm.cluster.Spec.KubernetesMasterName),
			})
			if rt != api.DryRun {
				cm.deleteDroplet(id)
				if cm.cluster.Spec.MasterReservedIP != "" {
					backoff.Retry(func() error {
						return cm.releaseReservedIP(cm.cluster.Spec.MasterReservedIP)
					}, backoff.NewExponentialBackOff())
				}
			}
		}

	}

	for _, node := range nodeGroups {
		if node.IsMaster() {
			continue
		}
		igm := &NodeGroupManager{
			cm: cm,
			instance: Instance{
				Type: InstanceType{
					Sku:          node.Spec.Template.Spec.SKU,
					Master:       false,
					SpotInstance: false,
				},
				Stats: GroupStats{
					Count: node.Spec.Nodes,
				},
			},
			im: im,
		}
		if clusterDelete || node.DeletionTimestamp != nil {
			instanceGroupName := igm.cm.namer.GetNodeGroupName(igm.instance.Type.Sku)
			acts = append(acts, api.Action{
				Action:   api.ActionDelete,
				Resource: "Node Group",
				Message:  fmt.Sprintf("Node group %v  will be deleted", instanceGroupName),
			})
			if rt != api.DryRun {
				err = igm.deleteNodeGroup(igm.instance.Type.Sku)
				Store(cm.ctx).NodeGroups(cm.cluster.Name).Delete(node.Name)
			}
		} else {
			act, _ := igm.AdjustNodeGroup(rt)
			acts = append(acts, act...)
			if rt != api.DryRun {
				node.Status.Nodes = (int32)(node.Spec.Nodes)
				Store(cm.ctx).NodeGroups(cm.cluster.Name).UpdateStatus(node)
			}
		}

	}
	if clusterDelete && rt != api.DryRun {
		// Delete SSH key from DB
		if err := cm.deleteSSHKey(); err != nil {
		}

		if err := DeleteARecords(cm.ctx, cm.cluster); err != nil {
		}
	}
	if rt != api.DryRun {
		time.Sleep(1 * time.Minute)

		for _, ng := range nodeGroups {
			groupName := cm.namer.GetNodeGroupName(ng.Spec.Template.Spec.SKU)
			_, providerInstances, _ := im.GetNodeGroup(groupName)

			runningInstance := make(map[string]*api.Node)
			for _, node := range providerInstances {
				runningInstance[node.Name] = node
			}

			clusterInstance, _ := GetClusterIstance(cm.ctx, cm.cluster, groupName)
			for _, node := range clusterInstance {
				fmt.Println(node)
				if _, found := runningInstance[node]; !found {
					err = DeleteClusterInstance(cm.ctx, cm.cluster, node)
					fmt.Println(err)
				}
			}
		}

		if !clusterDelete {
			cm.cluster.Status.Phase = api.ClusterReady
		} else {
			cm.cluster.Status.Phase = api.ClusterDeleted
		}
		Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
		Store(cm.ctx).Clusters().Update(cm.cluster)
	}

	return
}

func (cm *ClusterManager) getPublicKey() (bool, error) {
	_, _, err := cm.conn.client.Keys.GetByFingerprint(gtx.TODO(), SSHKey(cm.ctx).OpensshFingerprint)
	if err != nil {
		return false, err
	}
	return true, nil
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

func (cm *ClusterManager) getTags() (bool, error) {
	tag := "KubernetesCluster:" + cm.cluster.Name
	_, _, err := cm.conn.client.Tags.Get(gtx.TODO(), tag)
	if err != nil {
		// Tag does not already exist
		return false, err
	}
	return true, nil
}

func (cm *ClusterManager) createTags() error {
	tag := "KubernetesCluster:" + cm.cluster.Name
	_, _, err := cm.conn.client.Tags.Create(gtx.TODO(), &godo.TagCreateRequest{
		Name: tag,
	})
	if err != nil {
		return err
	}
	Logger(cm.ctx).Infof("Tag %v created", tag)
	return nil
}

func (cm *ClusterManager) getReserveIP() (bool, error) {
	if cm.cluster.Spec.MasterReservedIP == "auto" {
		_, _, err := cm.conn.client.FloatingIPs.Get(gtx.TODO(), cm.cluster.Spec.MasterReservedIP)
		if err != nil {
			return false, nil
		}
		return true, nil
	}
	return false, nil
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
