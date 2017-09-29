package vultr

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	gv "github.com/JamesClonk/vultr/lib"
	"github.com/appscode/go/errors"
	api "github.com/appscode/pharmer/apis/v1alpha1"
	. "github.com/appscode/pharmer/cloud"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (cm *ClusterManager) Apply(in *api.Cluster, dryRun bool) error {
	var err error
	if dryRun {
		action, err := cm.apply(in, api.DryRun)
		if err != nil {
			return err
		}
		jm, err := json.Marshal(action)
		if err != nil {
			return err
		}
		fmt.Println(string(jm))
	} else {
		if _, err = cm.apply(in, api.StdRun); err != nil {
			return err
		}
	}
	return nil
}

func (cm *ClusterManager) apply(in *api.Cluster, rt api.RunType) (acts []api.Action, err error) {
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

	defer func() {
		Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
	}()
	if err = cm.conn.detectInstanceImage(); err != nil {
		cm.cluster.Status.Reason = err.Error()
		errors.FromErr(err).WithContext(cm.ctx).Err()
		return
	}
	Logger(cm.ctx).Infof("Found vultr instance image %v", cm.cluster.Spec.Cloud.InstanceImage)

	nodeGroups, err := Store(cm.ctx).NodeGroups(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return
	}
	if in.DeletionTimestamp != nil && in.Status.Phase != api.ClusterDeleted {
		//clusterDelete = true
		if acts, err = cm.Delete(rt); err != nil {
			return
		}
	}
	if cm.cluster.Status.Phase == api.ClusterPending {
		// create cluster (master instance)
		var masterNG *api.NodeGroup
		var totalNodes int64 = 0
		for _, ng := range nodeGroups {
			if ng.IsMaster() {
				masterNG = ng
			} else {
				totalNodes += ng.Spec.Nodes
			}
		}
		cm.cluster.Spec.MasterSKU = "93"
		if totalNodes > 5 {
			cm.cluster.Spec.MasterSKU = "94"
		}
		// Master instance, reserve ip create
		if acts, err = cm.createCluster(rt); err != nil {
			return
		}
		masterNG.Status.Nodes = int32(1)
		Store(cm.ctx).NodeGroups(cm.cluster.Name).Update(masterNG)
	}

	im := &instanceManager{ctx: cm.ctx, cluster: cm.cluster, conn: cm.conn, namer: cm.namer}

	if cm.cluster.Status.Phase == api.ClusterReady || cm.cluster.Status.Phase == api.ClusterPending {
		// adjust node groups
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
			if node.DeletionTimestamp != nil {
				instanceGroupName := igm.cm.namer.GetNodeGroupName(igm.instance.Type.Sku)
				acts = append(acts, api.Action{
					Action:   api.ActionDelete,
					Resource: "Node Group",
					Message:  fmt.Sprintf("Node group %v  will be deleted", instanceGroupName),
				})
				if rt != api.DryRun {
					if err = igm.deleteNodeGroup(igm.instance.Type.Sku); err != nil {
						return
					}
					Store(cm.ctx).NodeGroups(cm.cluster.Name).Delete(node.Name)
				}
			} else {
				act, er := igm.AdjustNodeGroup(rt)
				if er != nil {
					err = er
					return
				}
				acts = append(acts, act...)
				if rt != api.DryRun {
					node.Status.Nodes = (int32)(node.Spec.Nodes)
					Store(cm.ctx).NodeGroups(cm.cluster.Name).UpdateStatus(node)
				}
			}

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
					if err = DeleteClusterInstance(cm.ctx, cm.cluster, node); err != nil {
						return
					}
				}
			}
		}

		if in.DeletionTimestamp != nil {
			cm.cluster.Status.Phase = api.ClusterDeleted
		} else {
			cm.cluster.Status.Phase = api.ClusterReady
		}
		Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
		Store(cm.ctx).Clusters().Update(cm.cluster)
	}
	return
}

func (cm *ClusterManager) createCluster(rt api.RunType) (acts []api.Action, err error) {
	im := &instanceManager{ctx: cm.ctx, cluster: cm.cluster, conn: cm.conn, namer: cm.namer}
	acts = make([]api.Action, 0)
	if found, _, _ := im.getPublicKey(); !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Public key",
			Message:  "Public key will be imported",
		})
		if rt != api.DryRun {
			if cm.cluster.Status.SSHKeyExternalID, err = cm.importPublicKey(); err != nil {
				cm.cluster.Status.Reason = err.Error()
				errors.FromErr(err).WithContext(cm.ctx).Err()
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Public key",
			Message:  "Public key found",
		})
	}

	if found, _ := cm.getReserveIP(); !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Reserve IP",
			Message:  "Reserved IP",
		})
		if rt != api.DryRun {
			if err = cm.reserveIP(); err != nil {
				cm.cluster.Status.Reason = err.Error()
				errors.FromErr(err).WithContext(cm.ctx).Err()
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Reserve IP",
			Message:  "Found reserve IP",
		})
	}

	var masterScriptId int
	var found bool
	if found, masterScriptId, _ = im.getStartupScript(cm.cluster.Spec.MasterSKU, api.RoleMaster); !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Master startup script",
			Message:  "Startup script will be created for masater instance",
		})
		if rt != api.DryRun {
			if masterScriptId, err = im.createStartupScript(cm.cluster.Spec.MasterSKU, api.RoleMaster); err != nil {
				cm.cluster.Status.Reason = err.Error()
				errors.FromErr(err).WithContext(cm.ctx).Err()
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Master startup script",
			Message:  "Startup script for masater instance found",
		})
	}

	var masterID string
	var masterServer *gv.Server
	if masterID, _ = im.getInstance(cm.cluster.Spec.KubernetesMasterName); masterID == "" {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Master instance",
			Message:  fmt.Sprintf("Master instance %v will be created", cm.cluster.Spec.KubernetesMasterName),
		})
		if rt != api.DryRun {
			masterID, err = im.createInstance(cm.cluster.Spec.KubernetesMasterName, cm.cluster.Spec.MasterSKU, masterScriptId)
			if err != nil {
				cm.cluster.Status.Reason = err.Error()
				errors.FromErr(err).WithContext(cm.ctx).Err()
				return
			}
			masterServer, err = cm.conn.waitForActiveInstance(masterID)
			if err != nil {
				cm.cluster.Status.Reason = err.Error()
				errors.FromErr(err).WithContext(cm.ctx).Err()
				return
			}
			if cm.cluster.Spec.MasterReservedIP != "" {
				err = im.assignReservedIP(cm.cluster.Spec.MasterReservedIP, masterID)
				if err != nil {
					cm.cluster.Status.Reason = err.Error()
					errors.FromErr(err).WithContext(cm.ctx).Err()
					return
				}
			}
		}

	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Master instance",
			Message:  fmt.Sprintf("Master instance %v found", cm.cluster.Spec.KubernetesMasterName),
		})
		if rt != api.DryRun {
			if masterServer, err = cm.conn.getServer(masterID); err != nil {
				return
			}
		}
	}
	if rt == api.DryRun {
		return
	}
	masterInstance, err := im.newKubeInstance(masterServer)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		errors.FromErr(err).WithContext(cm.ctx).Err()
		return
	}
	masterInstance.Spec.Role = api.RoleMaster
	cm.cluster.Spec.MasterExternalIP = masterInstance.Status.PublicIP
	cm.cluster.Spec.MasterInternalIP = masterInstance.Status.PrivateIP
	fmt.Println("Master EXTERNAL_IP", cm.cluster.Spec.MasterExternalIP, " --- Master INTERNAL_IP", cm.cluster.Spec.MasterInternalIP)
	Store(cm.ctx).Instances(cm.cluster.Name).Create(masterInstance)

	err = EnsureARecord(cm.ctx, cm.cluster, masterInstance) // works for reserved or non-reserved mode
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		errors.FromErr(err).WithContext(cm.ctx).Err()
		return
	}

	// Wait for master A record to propagate
	if err = EnsureDnsIPLookup(cm.ctx, cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		errors.FromErr(err).WithContext(cm.ctx).Err()
		return
	}

	// ----------------------------------------------------------------------------------
	// reboot master to use cert with internal_ip as SANS
	time.Sleep(60 * time.Second)
	Logger(cm.ctx).Info("Rebooting master instance")
	if err = im.reboot(masterID); err != nil {
		cm.cluster.Status.Reason = err.Error()
		errors.FromErr(err).WithContext(cm.ctx).Err()
		return
	}
	Logger(cm.ctx).Info("Rebooted master instance")
	// wait for nodes to start
	if err = WaitForReadyMaster(cm.ctx, cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		errors.FromErr(err).WithContext(cm.ctx).Err()
		return
	}

	cm.cluster.Status.Phase = api.ClusterReady
	// needed to get master_internal_ip
	if _, err = Store(cm.ctx).Clusters().UpdateStatus(cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return acts, err
	}
	return
}

func (cm *ClusterManager) importPublicKey() (string, error) {
	Logger(cm.ctx).Infof("Adding SSH public key")
	resp, err := cm.conn.client.CreateSSHKey(cm.cluster.Status.SSHKeyExternalID, string(SSHKey(cm.ctx).PublicKey))
	if err != nil {
		return "", errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	Logger(cm.ctx).Debugln("DO response", resp, " errors", err)
	Logger(cm.ctx).Infof("New ssh key with name %v and id %v created", cm.cluster.Status.SSHKeyExternalID, resp.ID)
	return resp.Name, nil
}

func (cm *ClusterManager) getReserveIP() (bool, error) {
	if cm.cluster.Spec.MasterReservedIP == "auto" {
		ips, err := cm.conn.client.ListReservedIP()
		if err != nil {
			return false, err
		}
		for _, ip := range ips {
			if ip.Label == cm.namer.ReserveIPName() {
				return true, nil
			}
		}
	}
	return false, nil
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
