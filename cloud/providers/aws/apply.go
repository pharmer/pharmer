package aws

import (
	"encoding/json"
	"fmt"
	"time"

	semver "github.com/appscode/go-version"
	. "github.com/appscode/go/context"
	. "github.com/appscode/go/types"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	clusterapi_aws "github.com/pharmer/pharmer/apis/v1beta1/aws"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/clusterdeployer/clusterclient"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
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
	if cm.conn, err = NewConnector(cm.ctx, cm.cluster, cm.owner); err != nil {
		return nil, err
	}
	cm.conn.namer = cm.namer

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

			if _, err := Store(cm.ctx).Owner(cm.owner).Clusters().UpdateStatus(cm.cluster); err != nil {
				return nil, err
			}
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
		nodeGroups, err := Store(cm.ctx).Owner(cm.owner).MachineSet(cm.cluster.Name).List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, ng := range nodeGroups {
			ng.Spec.Replicas = Int32P(0)
			_, err := Store(cm.ctx).Owner(cm.owner).MachineSet(cm.cluster.Name).Update(ng)
			if err != nil {
				return nil, err
			}
		}
	}

	{
		if err := cm.applyScale(dryRun); err != nil {
			return nil, err
		}
	}

	if cm.cluster.DeletionTimestamp != nil && cm.cluster.Status.Phase != api.ClusterDeleted {
		a, err := cm.applyDelete(dryRun)
		if err != nil {
			return nil, err
		}
		acts = append(acts, a...)
	}

	return acts, nil
}

func (cm *ClusterManager) applyCreate(dryRun bool) (acts []api.Action, err error) {
	if err := cm.SetupCerts(); err != nil {
		return nil, err
	}

	var found bool

	if found, err = cm.conn.getIAMProfile(); err != nil {
		Logger(cm.ctx).Infoln(err)
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
		Logger(cm.ctx).Infoln(err)
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

	var vpcID string
	if vpcID, found, err = cm.conn.getVpc(); err != nil {
		Logger(cm.ctx).Infoln(err)
	}
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "VPC",
			Message:  "Not found, will be created new vpc",
		})
		if !dryRun {
			if vpcID, err = cm.conn.setupVpc(); err != nil {
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "VPC",
			Message:  fmt.Sprintf("Found vpc with id %v", vpcID),
		})
	}

	var publicSubnetID string
	if publicSubnetID, found, err = cm.conn.getSubnet(vpcID, "public"); err != nil {
		Logger(cm.ctx).Infoln(err)
	}
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Subnet",
			Message:  "Public Subnet will be added",
		})
		if !dryRun {
			if publicSubnetID, err = cm.conn.setupPublicSubnet(vpcID); err != nil {
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Subnet",
			Message:  fmt.Sprintf("Public Subnet found with id %v", publicSubnetID),
		})
	}

	var privateSubnetID string
	if privateSubnetID, found, err = cm.conn.getSubnet(vpcID, "private"); err != nil {
		Logger(cm.ctx).Infoln(err)
	}
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Subnet",
			Message:  "Private Subnet will be added",
		})
		if !dryRun {
			if privateSubnetID, err = cm.conn.setupPrivateSubnet(vpcID); err != nil {
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Subnet",
			Message:  fmt.Sprintf("Private Subnet found with id %v", privateSubnetID),
		})
	}

	var gatewayID string
	if gatewayID, found, err = cm.conn.getInternetGateway(vpcID); err != nil {
		Logger(cm.ctx).Infoln(err)
	}
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Internet Gateway",
			Message:  "Internet gateway will be added",
		})
		if !dryRun {
			if gatewayID, err = cm.conn.setupInternetGateway(vpcID); err != nil {
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

	var natID string
	if natID, found, err = cm.conn.getNatGateway(vpcID); err != nil {
		Logger(cm.ctx).Infoln(err)
	}
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "NAT Gateway",
			Message:  "NAT gateway will be added",
		})
		if !dryRun {
			if natID, err = cm.conn.setupNatGateway(publicSubnetID); err != nil {
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "NAT Gateway",
			Message:  "NAT gateway found",
		})
	}

	var publicRouteTableID, privateRouteTableID string
	if publicRouteTableID, found, err = cm.conn.getRouteTable("public", vpcID); err != nil {
		Logger(cm.ctx).Infoln(err)
	}
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Route table",
			Message:  "Route table will be created",
		})
		if !dryRun {
			if publicRouteTableID, err = cm.conn.setupRouteTable("public", vpcID, gatewayID, natID, publicSubnetID, privateSubnetID); err != nil {
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

	if privateRouteTableID, found, err = cm.conn.getRouteTable("private", vpcID); err != nil {
		Logger(cm.ctx).Infoln(err)
	}
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Route table",
			Message:  "Route table will be created",
		})
		if !dryRun {
			if privateRouteTableID, err = cm.conn.setupRouteTable("private", vpcID, gatewayID, natID, publicSubnetID, privateSubnetID); err != nil {
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

	if _, found, err = cm.conn.getSecurityGroupID(vpcID, cm.cluster.Spec.Config.Cloud.AWS.MasterSGName); err != nil {
		Logger(cm.ctx).Infoln(err)
	}
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Security group",
			Message:  fmt.Sprintf("Master security group %v and node security group %v will be created", cm.cluster.Spec.Config.Cloud.AWS.MasterSGName, cm.cluster.Spec.Config.Cloud.AWS.NodeSGName),
		})
		if !dryRun {
			if err = cm.conn.setupSecurityGroups(vpcID); err != nil {
				cm.cluster.Status.Reason = err.Error()
				err = errors.Wrap(err, ID(cm.ctx))
				return
			}
		}
	} else {
		if err = cm.conn.detectSecurityGroups(vpcID); err != nil {
			return
		}
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Security group",
			Message:  fmt.Sprintf("Found master security group %v and node security group %v", cm.cluster.Spec.Config.Cloud.AWS.MasterSGName, cm.cluster.Spec.Config.Cloud.AWS.NodeSGName),
		})
	}

	if found, err = cm.conn.getBastion(); err != nil {
		Logger(cm.ctx).Infoln(err)
	}
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Bastion",
			Message:  fmt.Sprintf("Bastion will be created"),
		})
		if !dryRun {
			if err = cm.conn.setupBastion(publicSubnetID); err != nil {
				cm.cluster.Status.Reason = err.Error()
				err = errors.Wrap(err, ID(cm.ctx))
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Bastion",
			Message:  fmt.Sprintf("Found Bastion"),
		})
	}

	if found, err = cm.conn.getLoadBalancer(); err != nil {
		Logger(cm.ctx).Infoln(err)
	}
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Load Banalcer",
			Message:  fmt.Sprintf("Load Banalcer will be created"),
		})
		if !dryRun {
			if err = cm.conn.setupLoadBalancer(publicSubnetID); err != nil {
				cm.cluster.Status.Reason = err.Error()
				err = errors.Wrap(err, ID(cm.ctx))
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Load Banalcer",
			Message:  fmt.Sprintf("Found Load Balancer"),
		})
	}

	clusterSpec, err := clusterapi_aws.ClusterConfigFromProviderSpec(cm.cluster.Spec.ClusterAPI.Spec.ProviderSpec)
	if err != nil {
		return
	}

	clusterSpec.NetworkSpec = clusterapi_aws.NetworkSpec{
		VPC: clusterapi_aws.VPCSpec{
			ID:                vpcID,
			CidrBlock:         cm.cluster.Spec.Config.Cloud.AWS.VpcCIDR,
			InternetGatewayID: StringP(gatewayID),
			Tags: clusterapi_aws.Map{
				"sigs.k8s.io/cluster-api-provider-aws/managed": "true",
			},
		},
		Subnets: clusterapi_aws.Subnets{
			{
				ID:               publicSubnetID,
				CidrBlock:        cm.cluster.Spec.Config.Cloud.AWS.PublicSubnetCIDR,
				AvailabilityZone: cm.cluster.Spec.Config.Cloud.Zone,
				IsPublic:         true,
				RouteTableID:     StringP(publicRouteTableID),
				NatGatewayID:     StringP(natID),
			},
			{
				ID:               privateSubnetID,
				CidrBlock:        cm.cluster.Spec.Config.Cloud.AWS.PrivateSubnetCIDR,
				AvailabilityZone: cm.cluster.Spec.Config.Cloud.Zone,
				IsPublic:         false,
				RouteTableID:     StringP(privateRouteTableID),
			},
		},
	}

	rawClusterSpec, err := clusterapi_aws.EncodeClusterSpec(clusterSpec)
	if err != nil {
		return
	}

	cm.cluster.Spec.ClusterAPI.Spec.ProviderSpec.Value = rawClusterSpec
	if _, err = Store(cm.ctx).Owner(cm.owner).Clusters().Update(cm.cluster); err != nil {
		return
	}

	if found, err = cm.conn.getMaster(); err != nil {
		Logger(cm.ctx).Infoln(err)
	}
	if !found {
		Logger(cm.ctx).Info("Creating master instance")
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "MasterInstance",
			Message:  fmt.Sprintf("Master instance %s will be created", cm.namer.MasterName()),
		})
		if !dryRun {
			var machines []*clusterv1.Machine
			machines, err = Store(cm.ctx).Owner(cm.owner).Machine(cm.cluster.Name).List(metav1.ListOptions{})
			if err != nil {
				err = errors.Wrap(err, ID(cm.ctx))
				return
			}

			masterMachine, err := api.GetMasterMachine(machines)
			if err != nil {
				return nil, err
			}

			machineSets, err := Store(cm.ctx).Owner(cm.owner).MachineSet(cm.cluster.Name).List(metav1.ListOptions{})
			if err != nil {
				return nil, err
			}
			totalNodes := NodeCount(machineSets)

			// https://github.com/kubernetes/kubernetes/blob/8eb75a5810cba92ccad845ca360cf924f2385881/cluster/aws/config-default.sh#L33
			sku := "m3.large"
			if totalNodes > 10 {
				sku = "m3.xlarge"
			}
			if totalNodes > 100 {
				sku = "m3.2xlarge"
			}
			if totalNodes > 250 {
				sku = "c4.4xlarge"
			}
			if totalNodes > 500 {
				sku = "c4.8xlarge"
			}

			var masterServer *api.NodeInfo
			masterServer, err = cm.conn.startMaster(masterMachine, sku, privateSubnetID)
			if err != nil {
				cm.cluster.Status.Reason = err.Error()
				err = errors.Wrap(err, ID(cm.ctx))
				return acts, err
			}

			nodeAddresses := make([]core.NodeAddress, 0)

			nodeAddresses = append(nodeAddresses, core.NodeAddress{
				Type:    core.NodeExternalDNS,
				Address: cm.cluster.Status.Cloud.AWS.LBDNS,
			})

			if err = cm.cluster.SetClusterApiEndpoints(nodeAddresses); err != nil {
				return nil, err
			}

			if _, err = Store(cm.ctx).Owner(cm.owner).Clusters().Update(cm.cluster); err != nil {
				return nil, err
			}

			// update master machine spec
			specConfig := clusterapi_aws.AWSMachineProviderSpec{
				AMI: clusterapi_aws.AWSResourceReference{
					ID: StringP(cm.cluster.Spec.Config.Cloud.InstanceImage),
				},
				InstanceType: sku,
			}
			rawSpec, err := clusterapi_aws.EncodeMachineSpec(&specConfig)
			if err != nil {
				return nil, err
			}
			masterMachine.Spec.ProviderSpec.Value = rawSpec

			// update master machine status
			statusConfig := clusterapi_aws.AWSMachineProviderStatus{
				InstanceID: StringP(masterServer.ExternalID),
			}

			rawStatus, err := clusterapi_aws.EncodeMachineStatus(&statusConfig)
			if err != nil {
				return nil, err
			}
			masterMachine.Status.ProviderStatus = rawStatus

			// update in pharmer file
			masterMachine, err = Store(cm.ctx).Owner(cm.owner).Machine(cm.cluster.Name).Update(masterMachine)
			if err != nil {
				return nil, errors.Wrap(err, "error updating master machine in pharmer storage")
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Master Instance",
			Message:  "master instance(s) already exist",
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

	ca, err := NewClusterApi(cm.ctx, cm.cluster, cm.owner, "cloud-provider-system", kc, cm.conn)
	if err != nil {
		return acts, err
	}

	if err := ca.Apply(ControllerManager); err != nil {
		return acts, err
	}

	// needed to get master_internal_ip
	cm.cluster.Status.Phase = api.ClusterReady
	if _, err = Store(cm.ctx).Owner(cm.owner).Clusters().UpdateStatus(cm.cluster); err != nil {
		return
	}

	return
}

func (cm *ClusterManager) applyScale(dryRun bool) error {
	Logger(cm.ctx).Infoln("scaling machine set")

	//var msc *clusterv1.MachineSet
	machineSets, err := Store(cm.ctx).Owner(cm.owner).MachineSet(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	bc, err := GetBooststrapClient(cm.ctx, cm.cluster, cm.owner)
	if err != nil {
		return err
	}
	var data []byte
	for _, machineSet := range machineSets {
		if machineSet.DeletionTimestamp != nil {
			machineSet.DeletionTimestamp = nil
			if data, err = json.Marshal(machineSet); err != nil {
				return err
			}

			if err = bc.Delete(string(data)); err != nil {
				return nil
			}
			if err = Store(cm.ctx).Owner(cm.owner).MachineSet(cm.cluster.Name).Delete(machineSet.Name); err != nil {
				return nil
			}
		}

		existingMachineSet, err := bc.GetMachineSets(bc.GetContextNamespace())
		if err != nil {
			return err
		}

		if data, err = json.Marshal(machineSet); err != nil {
			return err
		}
		found := false
		for _, ems := range existingMachineSet {
			if ems.Name == machineSet.Name {
				found = true
				if err = bc.Apply(string(data)); err != nil {
					return err
				}
				break
			}
		}

		if !found {
			if err = bc.CreateMachineSets([]*clusterv1.MachineSet{machineSet}, bc.GetContextNamespace()); err != nil {
				return err
			}
		}
	}

	return nil
}

func (cm *ClusterManager) applyDelete(dryRun bool) ([]api.Action, error) {
	var acts []api.Action

	Logger(cm.ctx).Infoln("deleting cluster")

	if cm.cluster.Status.Phase == api.ClusterReady {
		cm.cluster.Status.Phase = api.ClusterDeleting
	}
	if _, err := Store(cm.ctx).Owner(cm.owner).Clusters().UpdateStatus(cm.cluster); err != nil {
		return nil, err
	}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "MasterInstance",
		Message:  fmt.Sprintf("Will delete master instance with name %v-master", cm.cluster.Name),
	})
	if !dryRun {
		if err := cm.conn.deleteInstance("controlplane"); err != nil {
			Logger(cm.ctx).Infof("Failed to delete master instance. Reason: %s", err)
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "SSH Key",
		Message:  "SSH key will be deleted",
	})
	if !dryRun {
		if err := cm.conn.deleteSSHKey(); err != nil {
			Logger(cm.ctx).Infoln(err)
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "IAM role",
		Message:  "IAM role will be deleted",
	})
	if !dryRun {
		if err := cm.conn.deleteIAMProfile(); err != nil {
			return acts, errors.Wrap(err, fmt.Sprintf("error deleting IAM Profiles"))
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "Load Balancers",
		Message:  "Load Balancers will be deleted",
	})
	if !dryRun {
		if _, err := cm.conn.deleteLoadBalancer(); err != nil {
			return acts, errors.Wrap(err, "error deleting load balancer")
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "Bastion Instance",
		Message:  "Bastion Instance will be deleted",
	})
	if !dryRun {
		if err := cm.conn.deleteInstance("bastion"); err != nil {
			return acts, errors.Wrap(err, "error deleting bastion instance")
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "Security Group",
		Message:  "Security Group will be deleted",
	})

	clusterSpec, err := clusterapi_aws.ClusterConfigFromProviderSpec(cm.cluster.Spec.ClusterAPI.Spec.ProviderSpec)
	if err != nil {
		return acts, err
	}

	vpcID := clusterSpec.NetworkSpec.VPC.ID
	var natID string
	if clusterSpec.NetworkSpec.Subnets[0].NatGatewayID != nil {
		natID = *clusterSpec.NetworkSpec.Subnets[0].NatGatewayID
	} else {
		natID = *clusterSpec.NetworkSpec.Subnets[0].NatGatewayID
	}

	if !dryRun {
		if err := cm.conn.deleteSecurityGroup(vpcID); err != nil {
			return acts, errors.Wrap(err, "error deleting security group")
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "Route Table",
		Message:  "Route Table will be deleted",
	})
	if !dryRun {
		if err := cm.conn.deleteRouteTable(vpcID); err != nil {
			return acts, errors.Wrap(err, "error deleting route table")
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "NAT",
		Message:  "NAT will be deleted",
	})
	if !dryRun {
		if err := cm.conn.deleteNatGateway(natID); err != nil {
			return acts, errors.Wrap(err, "error deleting NAT")
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "Elastic IP",
		Message:  "Elastic IP will be deleted",
	})
	if !dryRun {
		if err := cm.conn.releaseReservedIP(); err != nil {
			return acts, errors.Wrap(err, "error deleting Elastic IP")
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "IngernetGateway",
		Message:  "IngernetGateway will be deleted",
	})
	if !dryRun {
		if err := cm.conn.deleteInternetGateway(vpcID); err != nil {
			return acts, errors.Wrap(err, "error deleting Interget Gateway")
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "Subnet",
		Message:  "Subnets will be deleted",
	})
	if !dryRun {
		if err := cm.conn.deleteSubnetID(vpcID); err != nil {
			return acts, errors.Wrap(err, "error deleting Subnets")
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "VPC",
		Message:  "VPC will be deleted",
	})
	if !dryRun {
		if err := cm.conn.deleteVpc(vpcID); err != nil {
			return acts, errors.Wrap(err, "error deleting VPC")
		}
	}

	return acts, nil
}

func (cm *ClusterManager) applyUpgrade(dryRun bool) (acts []api.Action, err error) {
	var kc kubernetes.Interface
	if kc, err = cm.GetAdminClient(); err != nil {
		return
	}

	var masterMachine *clusterv1.Machine
	masterName := fmt.Sprintf("%v-master", cm.cluster.Name)
	masterMachine, err = Store(cm.ctx).Owner(cm.owner).Machine(cm.cluster.Name).Get(masterName)
	if err != nil {
		return nil, err
	}

	masterMachine.Spec.Versions.ControlPlane = cm.cluster.Spec.Config.KubernetesVersion
	masterMachine.Spec.Versions.Kubelet = cm.cluster.Spec.Config.KubernetesVersion

	var bc clusterclient.Client
	bc, err = GetBooststrapClient(cm.ctx, cm.cluster, cm.owner)
	if err != nil {
		return nil, err
	}

	var data []byte
	if data, err = json.Marshal(masterMachine); err != nil {
		return
	}
	if err = bc.Apply(string(data)); err != nil {
		return
	}

	// Wait until master updated
	desiredVersion, err := semver.NewVersion(cm.cluster.ClusterConfig().KubernetesVersion)
	if err != nil {
		return
	}
	if err = WaitForReadyMasterVersion(cm.ctx, kc, desiredVersion); err != nil {
		return
	}
	// wait for nodes to start
	if err = WaitForReadyMaster(cm.ctx, kc); err != nil {
		return
	}

	var machineSets []*clusterv1.MachineSet
	machineSets, err = Store(cm.ctx).Owner(cm.owner).MachineSet(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return
	}

	for _, machineSet := range machineSets {
		machineSet.Spec.Template.Spec.Versions.Kubelet = cm.cluster.Spec.Config.KubernetesVersion
		if data, err = json.Marshal(machineSet); err != nil {
			return
		}
		if err = bc.Apply(string(data)); err != nil {
			return
		}
	}

	if !dryRun {
		cm.cluster.Status.Phase = api.ClusterReady
		if _, err = Store(cm.ctx).Owner(cm.owner).Clusters().UpdateStatus(cm.cluster); err != nil {
			return
		}
	}
	return
}
