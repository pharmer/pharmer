package linode

import (
	"fmt"
	"strconv"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/go/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
)

func (cm *ClusterManager) Create(req *proto.ClusterCreateRequest) error {
	err := cm.initContext(req)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ins, err = cloud.NewInstances(cm.ctx, cm.cluster)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
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
		cloud.Store(cm.ctx).Instances(cm.cluster.Name).SaveInstances(cm.ins.Instances)
		cloud.Logger(cm.ctx).Infof("Cluster %v is %v", cm.cluster.Name, cm.cluster.Status.Phase)
		if cm.cluster.Status.Phase != api.ClusterPhaseReady {
			cloud.Logger(cm.ctx).Infof("Cluster %v is deleting", cm.cluster.Name)
			cm.Delete(&proto.ClusterDeleteRequest{
				Name:              cm.cluster.Name,
				ReleaseReservedIp: releaseReservedIp,
			})
		}
	}(cm.cluster.Spec.MasterReservedIP == "auto")

	if err = cm.conn.detectInstanceImage(); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Debugln("Linode instance image", cm.cluster.Spec.InstanceImage)

	if err = cm.conn.detectKernel(); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Infof("Linode kernel %v found", cm.cluster.Spec.Kernel)

	// -------------------------------------------------------------------ASSETS
	im := &instanceManager{ctx: cm.ctx, cluster: cm.cluster, conn: cm.conn, namer: cm.namer}

	masterScriptId, err := im.createStackScript(cm.cluster.Spec.MasterSKU, api.RoleKubernetesMaster)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterLinodeId, masterLinodeConfigId, err := im.createInstance(cm.cluster.Spec.KubernetesMasterName, masterScriptId, cm.cluster.Spec.MasterSKU)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterLinode, err := cm.conn.waitForStatus(masterLinodeId, LinodeStatus_PoweredOff)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Debugln("Linode", masterLinodeId, "is powered off.")
	masterInstance, err := im.newKubeInstance(masterLinode)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterInstance.Name = cm.namer.MasterName()
	masterInstance.Spec.Role = api.RoleKubernetesMaster
	cm.cluster.Spec.MasterExternalIP = masterInstance.Status.ExternalIP
	cm.cluster.Spec.MasterInternalIP = masterInstance.Status.InternalIP
	cm.ins.Instances = append(cm.ins.Instances, masterInstance)
	fmt.Println("Master EXTERNAL_IP", cm.cluster.Spec.MasterExternalIP, " ----- Master INTERNAL_IP", cm.cluster.Spec.MasterInternalIP)

	if cm.ctx, err = cloud.GenClusterCerts(cm.ctx, cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = cloud.EnsureARecord(cm.ctx, cm.cluster, masterInstance) // works for reserved or non-reserved mode
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// needed to get master_internal_ip
	if _, err = cloud.Store(cm.ctx).Clusters().UpdateStatus(cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if api.UseFirebase() {
		cloud.SaveInstancesInFirebase(cm.cluster, cm.ins)
	}

	// reboot master to use cert with internal_ip as SANS
	err = im.bootToGrub2(masterLinodeId, masterLinodeConfigId, masterInstance.Name)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// -----------------------------------------------------------------------------------

	// start nodes
	type NodeInfo struct {
		nodeId   int
		configId int
		state    int
	}
	nodes := make([]*NodeInfo, 0)
	for _, ng := range req.NodeGroups {
		nodeScriptId, err := im.createStackScript(ng.Sku, api.RoleKubernetesPool)
		if err != nil {
			cm.cluster.Status.Reason = err.Error()
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		for i := int64(0); i < ng.Count; i++ {
			linodeId, configId, err := im.createInstance(cm.namer.GenNodeName(), nodeScriptId, ng.Sku)
			if err != nil {
				cm.cluster.Status.Reason = err.Error()
				return errors.FromErr(err).WithContext(cm.ctx).Err()
			}
			nodes = append(nodes, &NodeInfo{
				nodeId:   linodeId,
				configId: configId,
				state:    0,
			})
		}
	}

	/*
		Now, for each node,
		- start at state = 0
		- Wait for the node to become PoweredOff and set state = 1
		- For 60 seconds, after it becomes PoweredOff (2 iteration with 30 sec delay)
		- At state = 3, Boot the node
		- Done, when all nodes had a chance to Boot
	*/
	var done bool
	for true {
		time.Sleep(30 * time.Second)
		done = true
		for _, info := range nodes {
			if info.state == 0 {
				resp, err := cm.conn.client.Linode.List(info.nodeId)
				// ignore error, and try again
				if err == nil {
					linode := resp.Linodes[0]
					cloud.Logger(cm.ctx).Infof("Instance %v (%v) is %v", linode.Label, linode.LinodeId, statusString(linode.Status))
					if linode.Status == LinodeStatus_PoweredOff {
						info.state = 1
						// create node
						node, err := im.newKubeInstance(&linode)
						if err != nil {
							cm.cluster.Status.Reason = err.Error()
							return errors.FromErr(err).WithContext(cm.ctx).Err()
						}
						node.Name = cm.cluster.Name + "-node-" + strconv.Itoa(info.nodeId)
						node.Spec.Role = api.RoleKubernetesPool
						cm.ins.Instances = append(cm.ins.Instances, node)
						cloud.Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)

						if api.UseFirebase() {
							cloud.SaveInstancesInFirebase(cm.cluster, cm.ins)
						}
					}
				}
			} else {
				info.state++
			}
			if info.state == 3 {
				err = im.bootToGrub2(info.nodeId, info.configId, cm.cluster.Name+"-node-"+strconv.Itoa(info.nodeId))
				if err != nil {
					info.state-- // retry on error
				}
			}
			if info.state < 3 {
				done = false
			}
		}
		if done {
			break
		}
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
