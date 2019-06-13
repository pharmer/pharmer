package aws

import (
	"fmt"
	"time"

	"github.com/appscode/go/log"
	. "github.com/appscode/go/types"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	clusterapi_aws "github.com/pharmer/pharmer/apis/v1beta1/aws"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

const (
	preTagDelay = 5 * time.Second
)

func (cm *ClusterManager) PrepareCloud(dryRun bool) ([]api.Action, error) {
	var (
		acts                []api.Action
		err                 error
		vpcID               string
		publicSubnetID      string
		privateSubnetID     string
		gatewayID           string
		natID               string
		publicRouteTableID  string
		privateRouteTableID string
	)

	if acts, err = ensureIAMProfile(cm.conn, acts, dryRun); err != nil {
		return acts, errors.Wrap(err, "failed to ensure iam profile")
	}

	if acts, err = importPublicKey(cm.conn, acts, dryRun); err != nil {
		return acts, errors.Wrap(err, "failed to import public key")
	}

	if acts, vpcID, err = ensureVPC(cm.conn, acts, dryRun); err != nil {
		return acts, errors.Wrap(err, "failed to ensure vpc")
	}

	if acts, publicSubnetID, privateSubnetID, err = ensureSubnet(cm.conn, acts, dryRun); err != nil {
		return acts, errors.Wrap(err, "failed to ensure subnets")
	}

	if acts, gatewayID, err = ensureInternetGateway(cm.conn, acts, vpcID, dryRun); err != nil {
		return acts, errors.Wrap(err, "failed to ensure igw")
	}

	if acts, natID, err = ensureNatGateway(cm.conn, acts, vpcID, publicSubnetID, dryRun); err != nil {
		return acts, errors.Wrap(err, "failed to ensure nat")
	}

	if acts, publicRouteTableID, privateRouteTableID, err = ensureRouteTable(cm.conn, acts, vpcID, gatewayID, natID, publicSubnetID, privateSubnetID, dryRun); err != nil {
		return acts, errors.Wrap(err, "failed to ensure route table")
	}

	if acts, err = ensureSecurityGroup(cm.conn, acts, vpcID, dryRun); err != nil {
		return acts, errors.Wrap(err, "failed to ensure security group")
	}

	if acts, err = ensureBastion(cm.conn, acts, publicSubnetID, dryRun); err != nil {
		return acts, errors.Wrap(err, "failed to ensure bastion")
	}

	if acts, err = ensureLoadBalancer(cm.conn, acts, publicSubnetID, dryRun); err != nil {
		return acts, errors.Wrap(err, "failed to ensure load balancer")
	}

	clusterSpec, err := clusterapi_aws.ClusterConfigFromProviderSpec(cm.Cluster.Spec.ClusterAPI.Spec.ProviderSpec)
	if err != nil {
		return acts, errors.Wrap(err, "failed to decode provider spec")
	}

	clusterSpec.NetworkSpec = clusterapi_aws.NetworkSpec{
		VPC: clusterapi_aws.VPCSpec{
			ID:                vpcID,
			CidrBlock:         cm.Cluster.Spec.Config.Cloud.AWS.VpcCIDR,
			InternetGatewayID: StringP(gatewayID),
			Tags: clusterapi_aws.Map{
				"sigs.k8s.io/cluster-api-provider-aws/managed": "true",
			},
		},
		Subnets: clusterapi_aws.Subnets{
			{
				ID:               publicSubnetID,
				CidrBlock:        cm.Cluster.Spec.Config.Cloud.AWS.PublicSubnetCIDR,
				AvailabilityZone: cm.Cluster.Spec.Config.Cloud.Zone,
				IsPublic:         true,
				RouteTableID:     StringP(publicRouteTableID),
				NatGatewayID:     StringP(natID),
			},
			{
				ID:               privateSubnetID,
				CidrBlock:        cm.Cluster.Spec.Config.Cloud.AWS.PrivateSubnetCIDR,
				AvailabilityZone: cm.Cluster.Spec.Config.Cloud.Zone,
				IsPublic:         false,
				RouteTableID:     StringP(privateRouteTableID),
			},
		},
	}

	rawClusterSpec, err := clusterapi_aws.EncodeClusterSpec(clusterSpec)
	if err != nil {
		return acts, err
	}

	cm.Cluster.Spec.ClusterAPI.Spec.ProviderSpec.Value = rawClusterSpec
	if _, err = store.StoreProvider.Clusters().Update(cm.Cluster); err != nil {
		return acts, err
	}

	return acts, nil
}

func (cm *ClusterManager) GetMasterSKU(totalNodes int32) string {
	sku := "t2.large"

	if totalNodes > 10 {
		sku = "t2.xlarge"
	}
	if totalNodes > 100 {
		sku = "t2.2xlarge"
	}
	if totalNodes > 250 {
		sku = "c4.4xlarge"
	}
	if totalNodes > 500 {
		sku = "c4.8xlarge"
	}

	return sku
}

func (cm *ClusterManager) EnsureMaster(acts []api.Action, dryRun bool) ([]api.Action, error) {
	var found bool

	leaderMachine, err := GetLeaderMachine(cm.Cluster)
	if err != nil {
		return acts, errors.Wrap(err, "failed to get leader machine")
	}

	if found, err = cm.conn.getMaster(leaderMachine.Name); err != nil {
		log.Infoln(err)
	}
	if !found {
		log.Info("Creating master instance")
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "MasterInstance",
			Message:  fmt.Sprintf("Master instance %s will be created", leaderMachine.Name),
		})
		if !dryRun {
			script, err := RenderStartupScript(cm, leaderMachine, "", customTemplate)
			if err != nil {
				return nil, err
			}

			var privateSubnetID, vpcID string
			if vpcID, _, err = cm.conn.getVpc(); err != nil {
				log.Infoln(err)
			}

			if privateSubnetID, found, err = cm.conn.getSubnet(vpcID, "private"); err != nil {
				log.Infoln(err)
			}

			masterInstance, err := cm.conn.startMaster(leaderMachine, privateSubnetID, script)
			if err != nil {
				cm.Cluster.Status.Reason = err.Error()
				err = errors.Wrap(err, "")
				return acts, err
			}

			if _, err = store.StoreProvider.Clusters().Update(cm.Cluster); err != nil {
				return nil, err
			}

			// update master machine spec
			spec, err := clusterapi_aws.MachineConfigFromProviderSpec(leaderMachine.Spec.ProviderSpec)
			if err != nil {
				log.Infof("Error decoding provider spec for machine %q", leaderMachine.Name)
				return nil, err
			}

			spec.AMI = clusterapi_aws.AWSResourceReference{
				ID: StringP(cm.Cluster.Spec.Config.Cloud.InstanceImage),
			}

			rootDeviceSize, err := cm.conn.getInstanceRootDeviceSize(masterInstance)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get root device size for master instance")
			}

			spec.RootDeviceSize = *rootDeviceSize

			rawSpec, err := clusterapi_aws.EncodeMachineSpec(spec)
			if err != nil {
				return nil, err
			}
			leaderMachine.Spec.ProviderSpec.Value = rawSpec

			// update master machine status
			statusConfig := clusterapi_aws.AWSMachineProviderStatus{
				InstanceID: masterInstance.InstanceId,
			}

			rawStatus, err := clusterapi_aws.EncodeMachineStatus(&statusConfig)
			if err != nil {
				return nil, err
			}
			leaderMachine.Status.ProviderStatus = rawStatus

			// update in pharmer file
			leaderMachine, err = store.StoreProvider.Machine(cm.Cluster.Name).Update(leaderMachine)
			if err != nil {
				return nil, errors.Wrap(err, "error updating master machine in pharmer storage")
			}
		}
	}

	return append(acts, api.Action{
		Action:   api.ActionNOP,
		Resource: "Master Instance",
		Message:  "master instance(s) already exist",
	}), nil

}

func ensureIAMProfile(conn *cloudConnector, acts []api.Action, dryRun bool) ([]api.Action, error) {
	var (
		found bool
		err   error
	)

	if found, err = conn.getIAMProfile(); err != nil {
		log.Infoln(err)
	}

	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "IAM Profile",
			Message:  "IAM profile will be created",
		})
		if !dryRun {
			if err := conn.ensureIAMProfile(); err != nil {
				return acts, err
			}
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionNOP,
		Resource: "IAM Profile",
		Message:  "IAM profile found",
	})

	return acts, nil
}

func importPublicKey(conn *cloudConnector, acts []api.Action, dryRun bool) ([]api.Action, error) {
	var (
		err   error
		found bool
	)
	if found, err = conn.getPublicKey(); err != nil {
		log.Infoln(err)
	}

	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "PublicKey",
			Message:  "Public key will be imported",
		})
		if !dryRun {
			if err = conn.importPublicKey(); err != nil {
				return acts, err
			}
		}
	}
	acts = append(acts, api.Action{
		Action:   api.ActionNOP,
		Resource: "PublicKey",
		Message:  "Public key found",
	})

	return acts, nil
}

func ensureVPC(conn *cloudConnector, acts []api.Action, dryRun bool) ([]api.Action, string, error) {
	var (
		vpcID string
		found bool
		err   error
	)

	if vpcID, found, err = conn.getVpc(); err != nil {
		log.Infoln(err)
	}
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "VPC",
			Message:  "Not found, will be created new vpc",
		})
		if !dryRun {
			if vpcID, err = conn.setupVpc(); err != nil {
				return acts, "", err
			}
		}
	}
	acts = append(acts, api.Action{
		Action:   api.ActionNOP,
		Resource: "VPC",
		Message:  fmt.Sprintf("Found vpc with id %v", vpcID),
	})

	return acts, vpcID, nil
}

func ensureSubnet(conn *cloudConnector, acts []api.Action, dryRun bool) ([]api.Action, string, string, error) {
	var (
		publicSubnetID  string
		privateSubnetID string
		found           bool
		err             error
	)

	vpcID, _, err := conn.getVpc()
	if err != nil {
		return acts, "", "", errors.Wrapf(err, "vpc not found")
	}

	if publicSubnetID, found, err = conn.getSubnet(vpcID, "public"); err != nil {
		log.Infoln(err)
	}
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Subnet",
			Message:  "Public Subnet will be added",
		})
		if !dryRun {
			if publicSubnetID, err = conn.setupPublicSubnet(vpcID); err != nil {
				return acts, "", "", err
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Subnet",
			Message:  fmt.Sprintf("Public Subnet found with id %v", publicSubnetID),
		})
	}

	if privateSubnetID, found, err = conn.getSubnet(vpcID, "private"); err != nil {
		log.Infoln(err)
	}
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Subnet",
			Message:  "Private Subnet will be added",
		})
		if !dryRun {
			if privateSubnetID, err = conn.setupPrivateSubnet(vpcID); err != nil {
				return acts, "", "", err
			}
		}
	}
	return append(acts, api.Action{
		Action:   api.ActionNOP,
		Resource: "Subnet",
		Message:  fmt.Sprintf("Private Subnet found with id %v", privateSubnetID),
	}), publicSubnetID, privateSubnetID, nil
}

func ensureInternetGateway(conn *cloudConnector, acts []api.Action, vpcID string, dryRun bool) ([]api.Action, string, error) {
	var (
		found     bool
		err       error
		gatewayID string
	)

	if gatewayID, found, err = conn.getInternetGateway(vpcID); err != nil {
		log.Infoln(err)
	}
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Internet Gateway",
			Message:  "Internet gateway will be added",
		})
		if !dryRun {
			if _, err = conn.setupInternetGateway(vpcID); err != nil {
				return acts, "", err
			}
		}
	}
	return append(acts, api.Action{
		Action:   api.ActionNOP,
		Resource: "Internet Gateway",
		Message:  "Internet gateway found",
	}), gatewayID, nil
}

func ensureNatGateway(conn *cloudConnector, acts []api.Action, vpcID, publicSubnetID string, dryRun bool) ([]api.Action, string, error) {
	var (
		found bool
		err   error
		natID string
	)
	if natID, found, err = conn.getNatGateway(vpcID); err != nil {
		log.Infoln(err)
	}
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "NAT Gateway",
			Message:  "NAT gateway will be added",
		})
		if !dryRun {
			if _, err = conn.setupNatGateway(publicSubnetID); err != nil {
				return acts, "", err
			}
		}
	}
	return append(acts, api.Action{
		Action:   api.ActionNOP,
		Resource: "NAT Gateway",
		Message:  "NAT gateway found",
	}), natID, nil
}

func ensureRouteTable(conn *cloudConnector, acts []api.Action, vpcID, gatewayID, natID, publicSubnetID, privateSubnetID string, dryRun bool) ([]api.Action, string, string, error) {
	var (
		found bool
		err   error
	)
	var publicRouteTableID, privateRouteTableID string
	if publicRouteTableID, found, err = conn.getRouteTable("public", vpcID); err != nil {
		log.Infoln(err)
	}
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Route table",
			Message:  "Route table will be created",
		})
		if !dryRun {
			if publicRouteTableID, err = conn.setupRouteTable("public", vpcID, gatewayID, natID, publicSubnetID, privateSubnetID); err != nil {
				return acts, "", "", err
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Route table",
			Message:  "Route table found",
		})
	}

	if privateRouteTableID, found, err = conn.getRouteTable("private", vpcID); err != nil {
		log.Infoln(err)
	}
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Route table",
			Message:  "Route table will be created",
		})
		if !dryRun {
			if privateRouteTableID, err = conn.setupRouteTable("private", vpcID, gatewayID, natID, publicSubnetID, privateSubnetID); err != nil {
				return acts, "", "", err
			}
		}
	}
	return append(acts, api.Action{
		Action:   api.ActionNOP,
		Resource: "Route table",
		Message:  "Route table found",
	}), publicRouteTableID, privateRouteTableID, nil
}

func ensureSecurityGroup(conn *cloudConnector, acts []api.Action, vpcID string, dryRun bool) ([]api.Action, error) {
	var (
		found bool
		err   error
	)
	if _, found, err = conn.getSecurityGroupID(vpcID, conn.Cluster.Spec.Config.Cloud.AWS.MasterSGName); err != nil {
		log.Infoln(err)
	}
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Security group",
			Message:  fmt.Sprintf("Master security group %v and node security group %v will be created", conn.Cluster.Spec.Config.Cloud.AWS.MasterSGName, conn.Cluster.Spec.Config.Cloud.AWS.NodeSGName),
		})
		if !dryRun {
			if err = conn.setupSecurityGroups(vpcID); err != nil {
				conn.Cluster.Status.Reason = err.Error()
				err = errors.Wrap(err, "")
				return acts, err
			}
		}
	} else {
		if err = conn.detectSecurityGroups(vpcID); err != nil {
			return acts, err
		}
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Security group",
			Message:  fmt.Sprintf("Found master security group %v and node security group %v", conn.Cluster.Spec.Config.Cloud.AWS.MasterSGName, conn.Cluster.Spec.Config.Cloud.AWS.NodeSGName),
		})
	}
	return acts, nil
}

func ensureBastion(conn *cloudConnector, acts []api.Action, publicSubnetID string, dryRun bool) ([]api.Action, error) {
	var (
		found bool
		err   error
	)
	if found, err = conn.getBastion(); err != nil {
		log.Infoln(err)
	}
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Bastion",
			Message:  fmt.Sprintf("Bastion will be created"),
		})
		if !dryRun {
			if err = conn.setupBastion(publicSubnetID); err != nil {
				conn.Cluster.Status.Reason = err.Error()
				err = errors.Wrap(err, "")
				return acts, err
			}
		}
	}
	return append(acts, api.Action{
		Action:   api.ActionNOP,
		Resource: "Bastion",
		Message:  fmt.Sprintf("Found Bastion"),
	}), nil
}

func ensureLoadBalancer(conn *cloudConnector, acts []api.Action, publicSubnetID string, dryRun bool) ([]api.Action, error) {
	var (
		found bool
		err   error
	)
	var loadbalancerDNS string
	if loadbalancerDNS, err = conn.getLoadBalancer(); err != nil {
		log.Infoln(err)
	}
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Load Banalcer",
			Message:  fmt.Sprintf("Load Banalcer will be created"),
		})
		if !dryRun {
			if loadbalancerDNS, err = conn.setupLoadBalancer(publicSubnetID); err != nil {
				conn.Cluster.Status.Reason = err.Error()
				err = errors.Wrap(err, "")
				return acts, err
			}
		}
	}
	acts = append(acts, api.Action{
		Action:   api.ActionNOP,
		Resource: "Load Banalcer",
		Message:  fmt.Sprintf("Found Load Balancer"),
	})

	if loadbalancerDNS == "" {
		return nil, errors.New("load balancer dns can't be empty")
	}

	// update load balancer field
	conn.Cluster.Status.Cloud.LoadBalancer = api.LoadBalancer{
		DNS:  loadbalancerDNS,
		Port: api.DefaultKubernetesBindPort,
	}

	nodeAddresses := []corev1.NodeAddress{
		{
			Type:    corev1.NodeExternalIP,
			Address: conn.Cluster.Status.Cloud.LoadBalancer.IP,
		},
	}

	if err = conn.Cluster.SetClusterApiEndpoints(nodeAddresses); err != nil {
		return acts, errors.Wrap(err, "Error setting controlplane endpoints")
	}
	return acts, nil
}

//func (cm *ClusterManager) ApplyCreate(dryRun bool) (acts []api.Action, leaderMachine *clusterv1.Machine, machines []*clusterv1.Machine, err error) {
//
//	machines, err = store.StoreProvider.Machine(cm.Cluster.Name).List(metav1.ListOptions{})
//	if err != nil {
//		err = errors.Wrap(err, "")
//		return
//	}
//
//	leaderMachine, err = GetLeaderMachine(cm.Cluster)
//	if err != nil {
//		return nil, nil, nil, err
//	}
//
//	machineSets, err := store.StoreProvider.MachineSet(cm.Cluster.Name).List(metav1.ListOptions{})
//	if err != nil {
//		return nil, nil, nil, err
//	}
//	totalNodes := NodeCount(machineSets)
//
//	// https://github.com/kubernetes/kubernetes/blob/8eb75a5810cba92ccad845ca360cf924f2385881/cluster/aws/config-default.sh#L33
//	sku := "t2.large"
//	if totalNodes > 10 {
//		sku = "t2.xlarge"
//	}
//	if totalNodes > 100 {
//		sku = "t2.2xlarge"
//	}
//	if totalNodes > 250 {
//		sku = "c4.4xlarge"
//	}
//	if totalNodes > 500 {
//		sku = "c4.8xlarge"
//	}
//
//	// update master machine spec
//	for _, m := range machines {
//		spec, err := clusterapi_aws.MachineConfigFromProviderSpec(m.Spec.ProviderSpec)
//		if err != nil {
//			log.Infof("Error decoding provider spec for machine %q", m.Name)
//			return nil, nil, nil, err
//		}
//
//		spec.InstanceType = sku
//
//		rawSpec, err := clusterapi_aws.EncodeMachineSpec(spec)
//		if err != nil {
//			return nil, nil, nil, err
//		}
//		m.Spec.ProviderSpec.Value = rawSpec
//
//		_, err = store.StoreProvider.Machine(cm.Cluster.Name).Update(m)
//		if err != nil {
//			return nil, nil, nil, errors.Wrapf(err, "failed to update machine %q to store", m.Name)
//		}
//	}
//
//	return
//}

func (cm *ClusterManager) ApplyDelete(dryRun bool) ([]api.Action, error) {
	var acts []api.Action

	log.Infoln("deleting cluster")

	if cm.Cluster.Status.Phase == api.ClusterReady {
		cm.Cluster.Status.Phase = api.ClusterDeleting
	}
	if _, err := store.StoreProvider.Clusters().UpdateStatus(cm.Cluster); err != nil {
		return nil, err
	}

	err := DeleteAllWorkerMachines(cm)
	if err != nil {
		log.Infof("failed to delete nodes: %v", err)
	}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "MasterInstance",
		Message:  fmt.Sprintf("Will delete master instance with name %v-master", cm.Cluster.Name),
	})
	if !dryRun {
		if err := cm.conn.deleteInstance("controlplane"); err != nil {
			log.Infof("Failed to delete master instance. Reason: %s", err)
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "SSH Key",
		Message:  "SSH key will be deleted",
	})
	if !dryRun {
		if err := cm.conn.deleteSSHKey(); err != nil {
			log.Infoln(err)
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

	clusterSpec, err := clusterapi_aws.ClusterConfigFromProviderSpec(cm.Cluster.Spec.ClusterAPI.Spec.ProviderSpec)
	if err != nil {
		return acts, err
	}

	vpcID, found, err := cm.conn.getVpc()
	if !found {
		log.Infof("vpc already deleted")
		return acts, nil
	}

	var natID string
	if len(clusterSpec.NetworkSpec.Subnets) > 0 {
		if clusterSpec.NetworkSpec.Subnets[0].NatGatewayID != nil {
			natID = *clusterSpec.NetworkSpec.Subnets[0].NatGatewayID
		} else {
			natID = *clusterSpec.NetworkSpec.Subnets[0].NatGatewayID
		}
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
