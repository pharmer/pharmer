package linode

import (
	"fmt"
	"strconv"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud/lib"
	"github.com/appscode/pharmer/storage"
	"github.com/appscode/pharmer/system"
)

func (cm *clusterManager) create(req *proto.ClusterCreateRequest) error {
	err := cm.initContext(req)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ins, err = lib.NewInstances(cm.ctx)
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
		cm.ctx.Logger.Infof("Cluster %v is %v", cm.ctx.Name, cm.ctx.Status)
		if cm.ctx.Status != storage.KubernetesStatus_Ready {
			cm.ctx.Logger.Infof("Cluster %v is deleting", cm.ctx.Name)
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
	cm.ctx.Logger.Debugln("Linode instance image", cm.ctx.InstanceImage)

	if err = cm.conn.detectKernel(); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger.Infof("Linode kernel %v found", cm.ctx.Kernel)

	// -------------------------------------------------------------------ASSETS
	im := &instanceManager{ctx: cm.ctx, conn: cm.conn, namer: cm.namer}

	masterScriptId, err := im.createStackScript(cm.ctx.MasterSKU, system.RoleKubernetesMaster)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterLinodeId, masterLinodeConfigId, err := im.createInstance(cm.ctx.KubernetesMasterName, masterScriptId, cm.ctx.MasterSKU)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterLinode, err := cm.conn.waitForStatus(masterLinodeId, LinodeStatus_PoweredOff)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger.Debugln("Linode", masterLinodeId, "is powered off.")
	masterInstance, err := im.newKubeInstance(masterLinode)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterInstance.Name = cm.namer.MasterName()
	masterInstance.Role = system.RoleKubernetesMaster
	cm.ctx.MasterExternalIP = masterInstance.ExternalIP
	cm.ctx.MasterInternalIP = masterInstance.InternalIP
	cm.ins.Instances = append(cm.ins.Instances, masterInstance)
	fmt.Println("Master EXTERNAL_IP", cm.ctx.MasterExternalIP, " ----- Master INTERNAL_IP", cm.ctx.MasterInternalIP)

	if err = lib.GenClusterCerts(cm.ctx); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = lib.EnsureARecord(cm.ctx, masterInstance) // works for reserved or non-reserved mode
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
	if api.UseFirebase() {
		lib.SaveInstancesInFirebase(cm.ctx.NewScriptOptions(), cm.ins)
	}

	// reboot master to use cert with internal_ip as SANS
	err = im.bootToGrub2(masterLinodeId, masterLinodeConfigId, masterInstance.Name)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
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
		nodeScriptId, err := im.createStackScript(ng.Sku, system.RoleKubernetesPool)
		if err != nil {
			cm.ctx.StatusCause = err.Error()
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		for i := int64(0); i < ng.Count; i++ {
			linodeId, configId, err := im.createInstance(cm.namer.GenNodeName(), nodeScriptId, ng.Sku)
			if err != nil {
				cm.ctx.StatusCause = err.Error()
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
					cm.ctx.Logger.Infof("Instance %v (%v) is %v", linode.Label, linode.LinodeId, statusString(linode.Status))
					if linode.Status == LinodeStatus_PoweredOff {
						info.state = 1
						// create node
						node, err := im.newKubeInstance(&linode)
						if err != nil {
							cm.ctx.StatusCause = err.Error()
							return errors.FromErr(err).WithContext(cm.ctx).Err()
						}
						node.Name = cm.ctx.Name + "-node-" + strconv.Itoa(info.nodeId)
						node.Role = system.RoleKubernetesPool
						cm.ins.Instances = append(cm.ins.Instances, node)
						cm.ctx.Save()

						cm.UploadStartupConfig()
						if api.UseFirebase() {
							lib.SaveInstancesInFirebase(cm.ctx.NewScriptOptions(), cm.ins)
						}
					}
				}
			} else {
				info.state++
			}
			if info.state == 3 {
				err = im.bootToGrub2(info.nodeId, info.configId, cm.ctx.Name+"-node-"+strconv.Itoa(info.nodeId))
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

	cm.ctx.Logger.Info("Waiting for cluster initialization")

	// Wait for master A record to propagate
	if err := lib.EnsureDnsIPLookup(cm.ctx); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// wait for nodes to start
	if err := lib.ProbeKubeAPI(cm.ctx); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// check all components are ok
	if err = lib.CheckComponentStatuses(cm.ctx); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// Make sure nodes are connected to master and are ready
	if err = lib.WaitForReadyNodes(cm.ctx); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	cm.ctx.Status = storage.KubernetesStatus_Ready
	return nil
}
