package vultr

import (
	"fmt"
	"strconv"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud/common"
	"github.com/appscode/pharmer/storage"
	"github.com/appscode/pharmer/system"
)

func (cm *clusterManager) create(req *proto.ClusterCreateRequest) error {
	err := cm.initContext(req)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ins, err = common.NewInstances(cm.ctx)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.conn, err = NewConnector(cm.ctx)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Save()

	defer func(releaseReservedIp bool) {
		if cm.ctx.Status == storage.KubernetesStatus_Pending {
			cm.ctx.Status = storage.KubernetesStatus_Failing
		}
		cm.ctx.Save()
		cm.ins.Save()
		cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, fmt.Sprintf("Cluster %v is %v", cm.ctx.Name, cm.ctx.Status))
		if cm.ctx.Status != storage.KubernetesStatus_Ready {
			cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, fmt.Sprintf("Cluster %v is deleting", cm.ctx.Name))
			cm.delete(&proto.ClusterDeleteRequest{
				Name:              cm.ctx.Name,
				ReleaseReservedIp: releaseReservedIp,
			})
		}
	}(cm.ctx.MasterReservedIP == "auto")

	if err = cm.conn.detectInstanceImage(); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, fmt.Sprintf("Found vultr instance image %v", cm.ctx.InstanceImage))

	cm.ctx.SSHKeyExternalID, err = cm.importPublicKey()
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = cm.reserveIP()
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// -------------------------------------------------------------------ASSETS
	im := &instanceManager{ctx: cm.ctx, conn: cm.conn, namer: cm.namer}

	masterScriptId, err := im.createStartupScript(cm.ctx.MasterSKU, system.RoleKubernetesMaster)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterID, err := im.createInstance(cm.ctx.KubernetesMasterName, cm.ctx.MasterSKU, masterScriptId)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterServer, err := cm.conn.waitForActiveInstance(masterID)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if cm.ctx.MasterReservedIP != "" {
		err = im.assignReservedIP(cm.ctx.MasterReservedIP, masterID)
		if err != nil {
			cm.ctx.StatusCause = err.Error()
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}
	masterInstance, err := im.newKubeInstance(masterServer)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterInstance.Role = system.RoleKubernetesMaster
	cm.ctx.MasterExternalIP = masterInstance.ExternalIP
	cm.ctx.MasterInternalIP = masterInstance.InternalIP
	fmt.Println("Master EXTERNAL_IP", cm.ctx.MasterExternalIP, " --- Master INTERNAL_IP", cm.ctx.MasterInternalIP)
	cm.ins.Instances = append(cm.ins.Instances, masterInstance)

	if err = common.GenClusterCerts(cm.ctx); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = common.EnsureARecord(cm.ctx, masterInstance) // works for reserved or non-reserved mode
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.DetectApiServerURL()
	// needed to get master_internal_ip
	if err = cm.ctx.Save(); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.UploadStartupConfig()

	// ----------------------------------------------------------------------------------
	// reboot master to use cert with internal_ip as SANS
	time.Sleep(60 * time.Second)
	cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, "Rebooting master instance")
	if err = im.reboot(masterID); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, "Rebooted master instance")
	// -----------------------------------------------------------------------------------

	// start nodes
	type NodeInfo struct {
		nodeId string
		state  int
	}
	nodes := make([]*NodeInfo, 0)
	for _, ng := range req.NodeGroups {
		nodeScriptId, err := im.createStartupScript(ng.Sku, system.RoleKubernetesPool)
		if err != nil {
			cm.ctx.StatusCause = err.Error()
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}

		for i := int64(0); i < ng.Count; i++ {
			nodeID, err := im.createInstance(cm.namer.GenNodeName(), ng.Sku, nodeScriptId)
			if err != nil {
				cm.ctx.StatusCause = err.Error()
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
						cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, fmt.Sprintf("Instance %v is %v", server.Name, server.Status))
						info.state = 1
						// create node
						node, err := im.newKubeInstance(&server)
						if err != nil {
							cm.ctx.StatusCause = err.Error()
							return errors.FromErr(err).WithContext(cm.ctx).Err()
						}
						node.Role = system.RoleKubernetesPool
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
	cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, "Waiting for cluster initialization")

	// Wait for master A record to propagate
	if err = common.EnsureDnsIPLookup(cm.ctx); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// wait for nodes to start
	if err = common.ProbeKubeAPI(cm.ctx); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// check all components are ok
	if err = common.CheckComponentStatuses(cm.ctx); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// Make sure nodes are connected to master and are ready
	if err = common.WaitForReadyNodes(cm.ctx); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	cm.ctx.Status = storage.KubernetesStatus_Ready
	return nil
}

func (cm *clusterManager) importPublicKey() (string, error) {
	cm.ctx.Logger().Infof("Adding SSH public key")
	resp, err := cm.conn.client.CreateSSHKey(cm.ctx.SSHKeyExternalID, string(cm.ctx.SSHKey.PublicKey))
	if err != nil {
		return "", errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().V(6).Infoln("DO response", resp, " errors", err)
	cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, fmt.Sprintf("New ssh key with name %v and id %v created", cm.ctx.SSHKeyExternalID, resp.ID))
	return resp.ID, nil
}

func (cm *clusterManager) reserveIP() error {
	if cm.ctx.MasterReservedIP == "auto" {
		regionID, err := strconv.Atoi(cm.ctx.Zone)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		ipID, err := cm.conn.client.CreateReservedIP(regionID, "v4", cm.namer.ReserveIPName())
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cm.ctx.Logger().V(6).Infoln("DO response", ipID, " errors", err)
		cm.ctx.Logger().Infof("Reserved new floating IP=%v", ipID)

		ip, err := cm.conn.client.GetReservedIP(ipID)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cm.ctx.MasterReservedIP = ip.Subnet
		cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, fmt.Sprintf("Floating ip %v reserved", ip.Subnet))
	}
	return nil
}
