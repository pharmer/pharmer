package vultr

import (
	"fmt"
	"strconv"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
)

func (cm *clusterManager) create(req *proto.ClusterCreateRequest) error {
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
	cm.ctx.Store().Clusters().SaveCluster(cm.cluster)

	defer func(releaseReservedIp bool) {
		if cm.cluster.Status.Phase == api.KubernetesStatus_Pending {
			cm.cluster.Status.Phase = api.KubernetesStatus_Failing
		}
		cm.ctx.Store().Clusters().SaveCluster(cm.cluster)
		cm.ctx.Store().Instances().SaveInstances(cm.ins.Instances)
		cm.ctx.Logger().Infof("Cluster %v is %v", cm.cluster.Name, cm.cluster.Status.Phase)
		if cm.cluster.Status.Phase != api.KubernetesStatus_Ready {
			cm.ctx.Logger().Infof("Cluster %v is deleting", cm.cluster.Name)
			cm.delete(&proto.ClusterDeleteRequest{
				Name:              cm.cluster.Name,
				ReleaseReservedIp: releaseReservedIp,
			})
		}
	}(cm.cluster.Spec.MasterReservedIP == "auto")

	if err = cm.conn.detectInstanceImage(); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Infof("Found vultr instance image %v", cm.cluster.Spec.InstanceImage)

	cm.cluster.Spec.SSHKeyExternalID, err = cm.importPublicKey()
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = cm.reserveIP()
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// -------------------------------------------------------------------ASSETS
	im := &instanceManager{ctx: cm.ctx, cluster: cm.cluster, conn: cm.conn, namer: cm.namer}

	masterScriptId, err := im.createStartupScript(cm.cluster.Spec.MasterSKU, api.RoleKubernetesMaster)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterID, err := im.createInstance(cm.cluster.Spec.KubernetesMasterName, cm.cluster.Spec.MasterSKU, masterScriptId)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterServer, err := cm.conn.waitForActiveInstance(masterID)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if cm.cluster.Spec.MasterReservedIP != "" {
		err = im.assignReservedIP(cm.cluster.Spec.MasterReservedIP, masterID)
		if err != nil {
			cm.cluster.Status.Reason = err.Error()
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}
	masterInstance, err := im.newKubeInstance(masterServer)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterInstance.Role = api.RoleKubernetesMaster
	cm.cluster.Spec.MasterExternalIP = masterInstance.ExternalIP
	cm.cluster.Spec.MasterInternalIP = masterInstance.InternalIP
	fmt.Println("Master EXTERNAL_IP", cm.cluster.Spec.MasterExternalIP, " --- Master INTERNAL_IP", cm.cluster.Spec.MasterInternalIP)
	cm.ins.Instances = append(cm.ins.Instances, masterInstance)

	if err = cloud.GenClusterCerts(cm.ctx, cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = cloud.EnsureARecord(cm.ctx, cm.cluster, masterInstance) // works for reserved or non-reserved mode
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.cluster.DetectApiServerURL()
	// needed to get master_internal_ip
	if err = cm.ctx.Store().Clusters().SaveCluster(cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.UploadStartupConfig()

	// ----------------------------------------------------------------------------------
	// reboot master to use cert with internal_ip as SANS
	time.Sleep(60 * time.Second)
	cm.ctx.Logger().Info("Rebooting master instance")
	if err = im.reboot(masterID); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Info("Rebooted master instance")
	// -----------------------------------------------------------------------------------

	// start nodes
	type NodeInfo struct {
		nodeId string
		state  int
	}
	nodes := make([]*NodeInfo, 0)
	for _, ng := range req.NodeGroups {
		nodeScriptId, err := im.createStartupScript(ng.Sku, api.RoleKubernetesPool)
		if err != nil {
			cm.cluster.Status.Reason = err.Error()
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}

		for i := int64(0); i < ng.Count; i++ {
			nodeID, err := im.createInstance(cm.namer.GenNodeName(), ng.Sku, nodeScriptId)
			if err != nil {
				cm.cluster.Status.Reason = err.Error()
				return errors.FromErr(err).WithContext(cm.ctx).Err()
			}
			nodes = append(nodes, &NodeInfo{
				nodeId: nodeID,
				state:  0,
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
				server, err := cm.conn.client.GetServer(info.nodeId)
				// ignore error, and try again
				if err == nil {
					cm.ctx.Logger().Infof("Instance %v (%v) is %v", server.Name, server.ID, server.Status)
					if server.Status == "active" && server.PowerStatus == "running" {
						cm.ctx.Logger().Infof("Instance %v is %v", server.Name, server.Status)
						info.state = 1
						// create node
						node, err := im.newKubeInstance(&server)
						if err != nil {
							cm.cluster.Status.Reason = err.Error()
							return errors.FromErr(err).WithContext(cm.ctx).Err()
						}
						node.Role = api.RoleKubernetesPool
						cm.ins.Instances = append(cm.ins.Instances, node)
					}
				}
			} else {
				info.state++
			}
			if info.state == 3 {
				err = cm.conn.client.RebootServer(info.nodeId)
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
	cm.ctx.Logger().Info("Waiting for cluster initialization")

	// Wait for master A record to propagate
	if err = cloud.EnsureDnsIPLookup(cm.ctx, cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// wait for nodes to start
	if err = cloud.ProbeKubeAPI(cm.ctx, cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// check all components are ok
	if err = cloud.CheckComponentStatuses(cm.ctx, cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// Make sure nodes are connected to master and are ready
	if err = cloud.WaitForReadyNodes(cm.ctx, cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	cm.cluster.Status.Phase = api.KubernetesStatus_Ready
	return nil
}

func (cm *clusterManager) importPublicKey() (string, error) {
	cm.ctx.Logger().Infof("Adding SSH public key")
	resp, err := cm.conn.client.CreateSSHKey(cm.cluster.Spec.SSHKeyExternalID, string(cm.cluster.Spec.SSHKey.PublicKey))
	if err != nil {
		return "", errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Debugln("DO response", resp, " errors", err)
	cm.ctx.Logger().Infof("New ssh key with name %v and id %v created", cm.cluster.Spec.SSHKeyExternalID, resp.ID)
	return resp.ID, nil
}

func (cm *clusterManager) reserveIP() error {
	if cm.cluster.Spec.MasterReservedIP == "auto" {
		regionID, err := strconv.Atoi(cm.cluster.Spec.Zone)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		ipID, err := cm.conn.client.CreateReservedIP(regionID, "v4", cm.namer.ReserveIPName())
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cm.ctx.Logger().Debugln("DO response", ipID, " errors", err)
		cm.ctx.Logger().Infof("Reserved new floating IP=%v", ipID)

		ip, err := cm.conn.client.GetReservedIP(ipID)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cm.cluster.Spec.MasterReservedIP = ip.Subnet
		cm.ctx.Logger().Infof("Floating ip %v reserved", ip.Subnet)
	}
	return nil
}
