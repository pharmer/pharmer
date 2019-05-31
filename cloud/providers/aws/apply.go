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
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

const (
	preTagDelay = 5 * time.Second
)

func (cm *ClusterManager) ApplyCreate(dryRun bool) (acts []api.Action, leaderMachine *clusterv1.Machine, machines []*clusterv1.Machine, err error) {
	var found bool

	if found, err = cm.conn.getIAMProfile(); err != nil {
		log.Infoln(err)
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
		log.Infoln(err)
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
		log.Infoln(err)
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
		log.Infoln(err)
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
		log.Infoln(err)
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
		log.Infoln(err)
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
		log.Infoln(err)
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
		log.Infoln(err)
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
		log.Infoln(err)
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

	if _, found, err = cm.conn.getSecurityGroupID(vpcID, cm.Cluster.Spec.Config.Cloud.AWS.MasterSGName); err != nil {
		log.Infoln(err)
	}
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Security group",
			Message:  fmt.Sprintf("Master security group %v and node security group %v will be created", cm.Cluster.Spec.Config.Cloud.AWS.MasterSGName, cm.Cluster.Spec.Config.Cloud.AWS.NodeSGName),
		})
		if !dryRun {
			if err = cm.conn.setupSecurityGroups(vpcID); err != nil {
				cm.Cluster.Status.Reason = err.Error()
				err = errors.Wrap(err, "")
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
			Message:  fmt.Sprintf("Found master security group %v and node security group %v", cm.Cluster.Spec.Config.Cloud.AWS.MasterSGName, cm.Cluster.Spec.Config.Cloud.AWS.NodeSGName),
		})
	}

	if found, err = cm.conn.getBastion(); err != nil {
		log.Infoln(err)
	}
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Bastion",
			Message:  fmt.Sprintf("Bastion will be created"),
		})
		if !dryRun {
			if err = cm.conn.setupBastion(publicSubnetID); err != nil {
				cm.Cluster.Status.Reason = err.Error()
				err = errors.Wrap(err, "")
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

	var loadbalancerDNS string
	if loadbalancerDNS, err = cm.conn.getLoadBalancer(); err != nil {
		log.Infoln(err)
	}
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Load Banalcer",
			Message:  fmt.Sprintf("Load Banalcer will be created"),
		})
		if !dryRun {
			if loadbalancerDNS, err = cm.conn.setupLoadBalancer(publicSubnetID); err != nil {
				cm.Cluster.Status.Reason = err.Error()
				err = errors.Wrap(err, "")
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

	if loadbalancerDNS == "" {
		return nil, leaderMachine, machines, errors.New("load balancer dns can't be empty")
	}

	// update load balancer field
	cm.Cluster.Status.Cloud.LoadBalancer = api.LoadBalancer{
		DNS:  loadbalancerDNS,
		Port: api.DefaultKubernetesBindPort,
	}

	clusterSpec, err := clusterapi_aws.ClusterConfigFromProviderSpec(cm.Cluster.Spec.ClusterAPI.Spec.ProviderSpec)
	if err != nil {
		return
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
		return
	}

	cm.Cluster.Spec.ClusterAPI.Spec.ProviderSpec.Value = rawClusterSpec
	if _, err = store.StoreProvider.Clusters().Update(cm.Cluster); err != nil {
		return
	}

	machines, err = store.StoreProvider.Machine(cm.Cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		err = errors.Wrap(err, "")
		return
	}

	leaderMachine, err = GetLeaderMachine(cm.Cluster)
	if err != nil {
		return nil, nil, nil, err
	}

	machineSets, err := store.StoreProvider.MachineSet(cm.Cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return nil, nil, nil, err
	}
	totalNodes := NodeCount(machineSets)

	// https://github.com/kubernetes/kubernetes/blob/8eb75a5810cba92ccad845ca360cf924f2385881/cluster/aws/config-default.sh#L33
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

	// update master machine spec
	for _, m := range machines {
		spec, err := clusterapi_aws.MachineConfigFromProviderSpec(m.Spec.ProviderSpec)
		if err != nil {
			log.Infof("Error decoding provider spec for machine %q", m.Name)
			return nil, nil, nil, err
		}

		spec.InstanceType = sku

		rawSpec, err := clusterapi_aws.EncodeMachineSpec(spec)
		if err != nil {
			return nil, nil, nil, err
		}
		m.Spec.ProviderSpec.Value = rawSpec

		_, err = store.StoreProvider.Machine(cm.Cluster.Name).Update(m)
		if err != nil {
			return nil, nil, nil, errors.Wrapf(err, "failed to update machine %q to store", m.Name)
		}
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
				return nil, nil, nil, err
			}

			masterInstance, err := cm.conn.startMaster(leaderMachine, sku, privateSubnetID, script)
			if err != nil {
				cm.Cluster.Status.Reason = err.Error()
				err = errors.Wrap(err, "")
				return acts, nil, nil, err
			}

			nodeAddresses := []core.NodeAddress{
				{
					Type:    core.NodeExternalDNS,
					Address: cm.Cluster.Status.Cloud.LoadBalancer.DNS,
				},
			}

			if err = cm.Cluster.SetClusterApiEndpoints(nodeAddresses); err != nil {
				return nil, nil, nil, err
			}

			if _, err = store.StoreProvider.Clusters().Update(cm.Cluster); err != nil {
				return nil, nil, nil, err
			}

			// update master machine spec
			spec, err := clusterapi_aws.MachineConfigFromProviderSpec(leaderMachine.Spec.ProviderSpec)
			if err != nil {
				log.Infof("Error decoding provider spec for machine %q", leaderMachine.Name)
				return nil, nil, nil, err
			}

			spec.AMI = clusterapi_aws.AWSResourceReference{
				ID: StringP(cm.Cluster.Spec.Config.Cloud.InstanceImage),
			}
			spec.InstanceType = sku

			rootDeviceSize, err := cm.conn.getInstanceRootDeviceSize(masterInstance)
			if err != nil {
				return nil, nil, nil, errors.Wrap(err, "failed to get root device size for master instance")
			}

			spec.RootDeviceSize = *rootDeviceSize

			rawSpec, err := clusterapi_aws.EncodeMachineSpec(spec)
			if err != nil {
				return nil, nil, nil, err
			}
			leaderMachine.Spec.ProviderSpec.Value = rawSpec

			// update master machine status
			statusConfig := clusterapi_aws.AWSMachineProviderStatus{
				InstanceID: masterInstance.InstanceId,
			}

			rawStatus, err := clusterapi_aws.EncodeMachineStatus(&statusConfig)
			if err != nil {
				return nil, nil, nil, err
			}
			leaderMachine.Status.ProviderStatus = rawStatus

			// update in pharmer file
			leaderMachine, err = store.StoreProvider.Machine(cm.Cluster.Name).Update(leaderMachine)
			if err != nil {
				return nil, nil, nil, errors.Wrap(err, "error updating master machine in pharmer storage")
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
