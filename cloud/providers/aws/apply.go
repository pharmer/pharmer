package aws

import (
	"fmt"
	"time"

	. "github.com/appscode/go/context"
	api "github.com/pharmer/pharmer/apis/v1alpha1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	preTagDelay = 5 * time.Second
)

func (cm *ClusterManager) Apply(in *api.Cluster, dryRun bool) ([]api.Action, error) {
	var err error
	var acts []api.Action

	if in.Status.Phase == "" {
		return nil, errors.Errorf("cluster `%s` is in unknown phase", cm.cluster.Name)
	}
	if in.Status.Phase == api.ClusterDeleted {
		return nil, nil
	}
	cm.cluster = in
	cm.namer = namer{cluster: cm.cluster}
	if cm.ctx, err = LoadCACertificates(cm.ctx, cm.cluster, cm.owner); err != nil {
		return nil, err
	}
	if cm.ctx, err = LoadSSHKey(cm.ctx, cm.cluster, cm.owner); err != nil {
		return nil, err
	}
	if cm.conn, err = NewConnector(cm.ctx, cm.cluster); err != nil {
		return nil, err
	}
	cm.conn.namer = cm.namer

	if err = cm.conn.detectUbuntuImage(); err != nil {
		return nil, err
	}
	cm.cluster.Spec.Cloud.InstanceImage = cm.conn.cluster.Spec.Cloud.InstanceImage

	if cm.cluster.Status.Phase == api.ClusterUpgrading {
		return nil, errors.Errorf("cluster `%s` is upgrading. Retry after cluster returns to Ready state", cm.cluster.Name)
	}
	if cm.cluster.Status.Phase == api.ClusterReady {
		var kc kubernetes.Interface
		kc, err = cm.GetAdminClient()
		if err != nil {
			return nil, err
		}
		if upgrade, err := NewKubeVersionGetter(kc, cm.cluster).IsUpgradeRequested(); err != nil {
			return nil, err
		} else if upgrade {
			cm.cluster.Status.Phase = api.ClusterUpgrading
			Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
			return cm.applyUpgrade(dryRun)
		}
	}

	if cm.cluster.Status.Phase == api.ClusterPending {
		a, err := cm.applyCreate(dryRun)
		if err != nil {
			return nil, err
		}
		acts = append(acts, a...)
	}

	if cm.cluster.DeletionTimestamp != nil && cm.cluster.Status.Phase != api.ClusterDeleted {
		nodeGroups, err := Store(cm.ctx).Owner(cm.owner).NodeGroups(cm.cluster.Name).List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, ng := range nodeGroups {
			ng.Spec.Nodes = 0
			_, err := Store(cm.ctx).Owner(cm.owner).NodeGroups(cm.cluster.Name).Update(ng)
			if err != nil {
				return nil, err
			}
		}
	}

	{
		a, err := cm.applyScale(dryRun)
		if err != nil {
			return nil, err
		}
		acts = append(acts, a...)
	}

	if cm.cluster.DeletionTimestamp != nil && cm.cluster.Status.Phase != api.ClusterDeleted {
		a, err := cm.applyDelete(dryRun)
		if err != nil {
			return nil, err
		}
		acts = append(acts, a...)
	}

	// detect-master
	// wait-master: via curl call polling
	// build-config

	//  # KUBE_SHARE_MASTER is used to add nodes to an existing master
	//  if [[ "${KUBE_SHARE_MASTER:-}" == "true" ]]; then
	//    detect-master
	//    start-nodes
	//    wait-nodes
	//  else
	//    start-master
	//    start-nodes
	//    wait-nodes
	//    wait-master
	//
	//    # Build ~/.kube/config
	//    build-config
	//  fi
	// check-cluster
	return acts, nil
}

func (cm *ClusterManager) applyCreate(dryRun bool) (acts []api.Action, err error) {
	var found bool
	// TODO: FixIt!
	//cm.cluster.Spec.RootDeviceName = cm.conn.cluster.Spec.RootDeviceName
	//fmt.Println(cm.cluster.Spec.Cloud.InstanceImage, cm.cluster.Spec.RootDeviceName, "---------------*********")

	if found, err = cm.conn.getIAMProfile(); err != nil {
		//return
	}
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "IAM Profile",
			Message:  "IAM profile will be created",
		})
		if !dryRun {
			if err = cm.conn.ensureIAMProfile(); err != nil {
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "IAM Profile",
			Message:  "IAM profile found",
		})
	}

	if found, err = cm.conn.getPublicKey(); err != nil {
		//return
	}

	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "PublicKey",
			Message:  "Public key will be imported",
		})
		if !dryRun {
			if err = cm.conn.importPublicKey(); err != nil {
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "PublicKey",
			Message:  "Public key found",
		})
	}

	if found, err = cm.conn.getVpc(); err != nil {
		//return
	}
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "VPC",
			Message:  "Not found, will be created new vpc",
		})
		if !dryRun {
			if err = cm.conn.setupVpc(); err != nil {
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "VPC",
			Message:  fmt.Sprintf("Found vpc with id %v", cm.cluster.Status.Cloud.AWS.VpcId),
		})
	}

	if found, err = cm.conn.getDHCPOptionSet(); err != nil {
		//return
	}
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "DSCP Option set",
			Message:  fmt.Sprintf("%v.compute.internal dscp option set will be created", cm.cluster.Spec.Cloud.Region),
		})
		if !dryRun {
			if err = cm.conn.createDHCPOptionSet(); err != nil {
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "DSCP Option set",
			Message:  fmt.Sprintf("Found %v.compute.internal dscp option set", cm.cluster.Spec.Cloud.Region),
		})
	}

	if found, err = cm.conn.getSubnet(); err != nil {
		//return
	}
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Subnet",
			Message:  "Subnet will be added",
		})
		if !dryRun {
			if err = cm.conn.setupSubnet(); err != nil {
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Subnet",
			Message:  fmt.Sprintf("Subnet found with id %v", cm.cluster.Status.Cloud.AWS.SubnetId),
		})
	}

	if found, err = cm.conn.getInternetGateway(); err != nil {
		//return
	}
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Internet Gateway",
			Message:  "Internet gateway will be added",
		})
		if !dryRun {
			if err = cm.conn.setupInternetGateway(); err != nil {
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Internet Gateway",
			Message:  "Internet gateway found",
		})
	}

	if found, err = cm.conn.getRouteTable(); err != nil {
		//return
	}
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Route table",
			Message:  "Route table will be created",
		})
		if !dryRun {
			if err = cm.conn.setupRouteTable(); err != nil {
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Route table",
			Message:  "Route table found",
		})
	}

	if _, found, err = cm.conn.getSecurityGroupId(cm.cluster.Spec.Cloud.AWS.MasterSGName); err != nil {
		//return
	}
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Security group",
			Message:  fmt.Sprintf("Master security group %v and node security group %v will be created", cm.cluster.Spec.Cloud.AWS.MasterSGName, cm.cluster.Spec.Cloud.AWS.NodeSGName),
		})
		if !dryRun {
			if err = cm.conn.setupSecurityGroups(); err != nil {
				cm.cluster.Status.Reason = err.Error()
				err = errors.Wrap(err, ID(cm.ctx))
				return
			}
		}
	} else {
		if err = cm.conn.detectSecurityGroups(); err != nil {
			return
		}
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Security group",
			Message:  fmt.Sprintf("Found master security group %v and node security group %v", cm.cluster.Spec.Cloud.AWS.MasterSGName, cm.cluster.Spec.Cloud.AWS.NodeSGName),
		})
	}

	var nodeGroups []*api.NodeGroup
	nodeGroups, err = Store(cm.ctx).Owner(cm.owner).NodeGroups(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		err = errors.Wrap(err, ID(cm.ctx))
		return
	}
	masterNG, err := FindMasterNodeGroup(nodeGroups)
	if err != nil {
		return
	}
	if masterNG.Spec.Template.Spec.SKU == "" {
		totalNodes := NodeCount(nodeGroups)
		// https://github.com/kubernetes/kubernetes/blob/8eb75a5810cba92ccad845ca360cf924f2385881/cluster/aws/config-default.sh#L33
		masterNG.Spec.Template.Spec.SKU = "m3.large"
		if totalNodes > 10 {
			masterNG.Spec.Template.Spec.SKU = "m3.xlarge"
		}
		if totalNodes > 100 {
			masterNG.Spec.Template.Spec.SKU = "m3.2xlarge"
		}
		if totalNodes > 250 {
			masterNG.Spec.Template.Spec.SKU = "c4.4xlarge"
		}
		if totalNodes > 500 {
			masterNG.Spec.Template.Spec.SKU = "c4.8xlarge"
		}
		masterNG, err = Store(cm.ctx).Owner(cm.owner).NodeGroups(cm.cluster.Name).Update(masterNG)
		if err != nil {
			return
		}
	}

	if found, err = cm.conn.getMaster(); err != nil {
		//return
	}
	if !found {
		Logger(cm.ctx).Info("Creating master instance")
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "MasterInstance",
			Message:  fmt.Sprintf("Master instance %s will be created", cm.namer.MasterName()),
		})
		if !dryRun {
			var masterServer *api.NodeInfo
			masterServer, err = cm.conn.startMaster(cm.namer.MasterName(), masterNG)
			if err != nil {
				cm.cluster.Status.Reason = err.Error()
				err = errors.Wrap(err, ID(cm.ctx))
				return acts, err
			}
			if masterServer.PrivateIP != "" {
				cm.cluster.Status.APIAddresses = append(cm.cluster.Status.APIAddresses, core.NodeAddress{
					Type:    core.NodeInternalIP,
					Address: masterServer.PrivateIP,
				})
			}

			var kc kubernetes.Interface
			kc, err = cm.GetAdminClient()
			if err != nil {
				return
			}
			// wait for nodes to start
			if err = WaitForReadyMaster(cm.ctx, kc); err != nil {
				return
			}

			masterNG.Status.Nodes = 1
			masterNG, err = Store(cm.ctx).Owner(cm.owner).NodeGroups(cm.cluster.Name).Update(masterNG)
			if err != nil {
				return
			}

			// needed to get master_internal_ip
			cm.cluster.Status.Phase = api.ClusterReady
			if _, err = Store(cm.ctx).Clusters().UpdateStatus(cm.cluster); err != nil {
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Master Instance",
			Message:  "master instance(s) already exist",
		})
	}

	return
}

func (cm *ClusterManager) applyScale(dryRun bool) (acts []api.Action, err error) {
	var nodeGroups []*api.NodeGroup
	nodeGroups, err = Store(cm.ctx).Owner(cm.owner).NodeGroups(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return
	}

	var token string
	var kc kubernetes.Interface
	if cm.cluster.Status.Phase != api.ClusterPending {
		kc, err = cm.GetAdminClient()
		if err != nil {
			return
		}
		if !dryRun {
			if token, err = GetExistingKubeadmToken(kc, api.TokenDuration_10yr); err != nil {
				return
			}
			if cm.cluster, err = Store(cm.ctx).Clusters().Update(cm.cluster); err != nil {
				return
			}
		}

	}
	for _, node := range nodeGroups {
		if node.IsMaster() {
			continue
		}
		igm := NewAWSNodeGroupManager(cm.ctx, cm.conn, cm.namer, node, kc, token)
		var a2 []api.Action
		a2, err = igm.Apply(dryRun)
		if err != nil {
			return
		}
		acts = append(acts, a2...)
	}

	Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
	Store(cm.ctx).Clusters().Update(cm.cluster)

	return
}

func (cm *ClusterManager) applyDelete(dryRun bool) (acts []api.Action, err error) {
	var found bool

	if cm.cluster.Status.Phase == api.ClusterReady {
		cm.cluster.Status.Phase = api.ClusterDeleting
	}
	_, err = Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
	if err != nil {
		return
	}

	var nodeGroups []*api.NodeGroup
	nodeGroups, err = Store(cm.ctx).Owner(cm.owner).NodeGroups(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return
	}
	var masterNG *api.NodeGroup
	masterNG, err = FindMasterNodeGroup(nodeGroups)
	if err != nil {
		return
	}

	var kc kubernetes.Interface
	kc, err = cm.GetAdminClient()
	if err != nil {
		return
	}

	found, err = cm.conn.findVPC()
	if err != nil {
		return
	}
	if !found {
		err = errors.Errorf("[%s] VPC %v not found for Cluster %v", ID(cm.ctx), cm.cluster.Status.Cloud.AWS.VpcId, cm.cluster.Name)
		return
	}

	var masterInstance *core.Node
	masterInstance, err = kc.CoreV1().Nodes().Get(cm.namer.MasterName(), metav1.GetOptions{})
	if err != nil && !kerr.IsNotFound(err) {
		return
	}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "Master Instance",
		Message:  "master instance(s) will be deleted",
	})
	if !dryRun {
		if err = cm.conn.deleteMaster(); err != nil {
			return
		}

		if err = cm.conn.ensureInstancesDeleted(); err != nil {
			//return
		}
	}
	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "Master Instance volume",
		Message:  "master instance(s) volume will be deleted",
	})
	if !dryRun {
		if err = cm.conn.deleteVolume(); err != nil {
			return
		}
	}

	if masterNG.Spec.Template.Spec.ExternalIPType == api.IPTypeReserved {
		for _, addr := range masterInstance.Status.Addresses {
			if addr.Type == core.NodeExternalIP {
				acts = append(acts, api.Action{
					Action:   api.ActionDelete,
					Resource: "Reserved IP",
					Message:  "Reserved IP will be released",
				})
				if !dryRun {
					if err = cm.conn.releaseReservedIP(addr.Address); err != nil {
						return
					}
				}
			}
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "Security Group",
		Message:  "Security Group will be deleted",
	})
	if !dryRun {
		if err = cm.conn.deleteSecurityGroup(); err != nil {
			return
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "Internet Gateway",
		Message:  "Internet gateway will be deleted",
	})
	if !dryRun {
		if err = cm.conn.deleteInternetGateway(); err != nil {
			return
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "DHCP Option",
		Message:  "DHCP option will be deleted",
	})
	if !dryRun {
		if err = cm.conn.deleteDHCPOption(); err != nil {
			return
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "Route Table",
		Message:  "master instance(s) volume will be deleted",
	})
	if !dryRun {
		if err = cm.conn.deleteRouteTable(); err != nil {
			return
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "Subnet ID",
		Message:  "Subnet id will be deleted",
	})
	if !dryRun {
		if err = cm.conn.deleteSubnetId(); err != nil {
			return
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "VPC",
		Message:  "VPC will be deleted",
	})
	if !dryRun {
		if err = cm.conn.deleteVpc(); err != nil {
			return
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "SSH Key",
		Message:  "SSH key will be deleted",
	})
	if !dryRun {
		if err = cm.conn.deleteSSHKey(); err != nil {
			return
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "IAM role",
		Message:  "IAM role will be deleted",
	})
	if !dryRun {
		if err = cm.conn.deleteIAMProfile(); err != nil {
			return
		}
	}

	if !dryRun {
		cm.cluster.Status.Phase = api.ClusterDeleted
		Store(cm.ctx).Clusters().Update(cm.cluster)
	}

	return
}

func (cm *ClusterManager) applyUpgrade(dryRun bool) (acts []api.Action, err error) {
	var kc kubernetes.Interface
	if kc, err = cm.GetAdminClient(); err != nil {
		return
	}

	upm := NewUpgradeManager(cm.ctx, cm, kc, cm.cluster, cm.owner)
	if !dryRun {
		var a []api.Action
		a, err = upm.Apply(dryRun)
		if err != nil {
			return
		}
		acts = append(acts, a...)
	}

	var nodeGroups []*api.NodeGroup
	if nodeGroups, err = Store(cm.ctx).Owner(cm.owner).NodeGroups(cm.cluster.Name).List(metav1.ListOptions{}); err != nil {
		return
	}

	var token string

	if !dryRun {
		if token, err = GetExistingKubeadmToken(kc, api.TokenDuration_10yr); err != nil {
			return
		}
		if cm.cluster, err = Store(cm.ctx).Clusters().Update(cm.cluster); err != nil {
			return
		}
	}

	for _, ng := range nodeGroups {
		if !ng.IsMaster() {
			acts = append(acts, api.Action{
				Action:   api.ActionUpdate,
				Resource: "Instance Template",
				Message:  fmt.Sprintf("Instance template of %v will be updated to %v", ng.Name, cm.namer.LaunchConfigName(ng.Spec.Template.Spec.SKU)),
			})
			if !dryRun {
				if err = cm.conn.updateLaunchConfigurationTemplate(ng, token); err != nil {
					return
				}
			}
		}
	}

	if !dryRun {
		cm.cluster.Status.Phase = api.ClusterReady
		if _, err = Store(cm.ctx).Clusters().UpdateStatus(cm.cluster); err != nil {
			return
		}
	}

	return
}
