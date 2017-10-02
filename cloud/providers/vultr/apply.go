package vultr

import (
	"fmt"
	"strconv"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/go/errors"
	"github.com/appscode/pharmer/api"
	. "github.com/appscode/pharmer/cloud"
)

func (cm *ClusterManager) Apply(in *api.Cluster, dryRun bool) ([]api.Action, error) {
	var err error

	cm.cluster = in
	if cm.conn, err = NewConnector(cm.ctx, cm.cluster); err != nil {
		return nil, err
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

	if err = cm.conn.detectInstanceImage(); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	Logger(cm.ctx).Infof("Found vultr instance image %v", cm.cluster.Spec.Cloud.InstanceImage)

	cm.cluster.Status.SSHKeyExternalID, err = cm.importPublicKey()
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = cm.reserveIP()
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// -------------------------------------------------------------------ASSETS
	im := &instanceManager{ctx: cm.ctx, cluster: cm.cluster, conn: cm.conn, namer: cm.namer}

	masterScriptId, err := im.createStartupScript(cm.cluster.Spec.MasterSKU, api.RoleMaster)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterID, err := im.createInstance(cm.cluster.Spec.KubernetesMasterName, cm.cluster.Spec.MasterSKU, masterScriptId)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterServer, err := cm.conn.waitForActiveInstance(masterID)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if cm.cluster.Spec.MasterReservedIP != "" {
		err = im.assignReservedIP(cm.cluster.Spec.MasterReservedIP, masterID)
		if err != nil {
			cm.cluster.Status.Reason = err.Error()
			return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}
	masterInstance, err := im.newKubeInstance(masterServer)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterInstance.Spec.Role = api.RoleMaster
	cm.cluster.Spec.MasterExternalIP = masterInstance.Status.PublicIP
	cm.cluster.Spec.MasterInternalIP = masterInstance.Status.PrivateIP
	fmt.Println("Master EXTERNAL_IP", cm.cluster.Spec.MasterExternalIP, " --- Master INTERNAL_IP", cm.cluster.Spec.MasterInternalIP)
	Store(cm.ctx).Instances(cm.cluster.Name).Create(masterInstance)

	err = EnsureARecord(cm.ctx, cm.cluster, masterInstance) // works for reserved or non-reserved mode
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// needed to get master_internal_ip
	if _, err = Store(cm.ctx).Clusters().UpdateStatus(cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// ----------------------------------------------------------------------------------
	// reboot master to use cert with internal_ip as SANS
	time.Sleep(60 * time.Second)
	Logger(cm.ctx).Info("Rebooting master instance")
	if err = im.reboot(masterID); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	Logger(cm.ctx).Info("Rebooted master instance")
	// -----------------------------------------------------------------------------------

	// start nodes
	type NodeInfo struct {
		nodeId string
		state  int
	}
	nodes := make([]*NodeInfo, 0)
	//for _, ng := range req.NodeGroups {
	//	nodeScriptId, err := im.createStartupScript(ng.Sku, api.RoleKubernetesPool)
	//	if err != nil {
	//		cm.cluster.Status.Reason = err.Error()
	//		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	//	}
	//
	//	for i := int64(0); i < ng.Count; i++ {
	//		nodeID, err := im.createInstance(cm.namer.GenNodeName(), ng.Sku, nodeScriptId)
	//		if err != nil {
	//			cm.cluster.Status.Reason = err.Error()
	//			return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	//		}
	//		nodes = append(nodes, &NodeInfo{
	//			nodeId: nodeID,
	//			state:  0,
	//		})
	//	}
	//}

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
					Logger(cm.ctx).Infof("Instance %v (%v) is %v", server.Name, server.ID, server.Status)
					if server.Status == "active" && server.PowerStatus == "running" {
						Logger(cm.ctx).Infof("Instance %v is %v", server.Name, server.Status)
						info.state = 1
						// create node
						node, err := im.newKubeInstance(&server)
						if err != nil {
							cm.cluster.Status.Reason = err.Error()
							return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
						}
						node.Spec.Role = api.RoleNode
						Store(cm.ctx).Instances(cm.cluster.Name).Create(node)
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
	Logger(cm.ctx).Info("Waiting for cluster initialization")

	// Wait for master A record to propagate
	if err = EnsureDnsIPLookup(cm.ctx, cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	kc, err := cm.GetAdminClient()
	// wait for nodes to start
	if err = WaitForReadyMaster(cm.ctx, kc); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.cluster.Status.Phase = api.ClusterReady
	return nil, nil
}

func (cm *ClusterManager) importPublicKey() (string, error) {
	Logger(cm.ctx).Infof("Adding SSH public key")
	resp, err := cm.conn.client.CreateSSHKey(cm.cluster.Status.SSHKeyExternalID, string(SSHKey(cm.ctx).PublicKey))
	if err != nil {
		return "", errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	Logger(cm.ctx).Debugln("DO response", resp, " errors", err)
	Logger(cm.ctx).Infof("New ssh key with name %v and id %v created", cm.cluster.Status.SSHKeyExternalID, resp.ID)
	return resp.ID, nil
}

func (cm *ClusterManager) reserveIP() error {
	if cm.cluster.Spec.MasterReservedIP == "auto" {
		regionID, err := strconv.Atoi(cm.cluster.Spec.Cloud.Zone)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		ipID, err := cm.conn.client.CreateReservedIP(regionID, "v4", cm.namer.ReserveIPName())
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		Logger(cm.ctx).Debugln("DO response", ipID, " errors", err)
		Logger(cm.ctx).Infof("Reserved new floating IP=%v", ipID)

		ip, err := cm.conn.client.GetReservedIP(ipID)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cm.cluster.Spec.MasterReservedIP = ip.Subnet
		Logger(cm.ctx).Infof("Floating ip %v reserved", ip.Subnet)
	}
	return nil
}
