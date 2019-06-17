package aws

import (
	"time"

	"github.com/appscode/go/log"
	"github.com/appscode/go/types"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	clusterapi_aws "github.com/pharmer/pharmer/apis/v1beta1/aws"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

const (
	preTagDelay = 5 * time.Second
)

func (cm *ClusterManager) PrepareCloud() error {
	var (
		err                 error
		vpcID               string
		publicSubnetID      string
		privateSubnetID     string
		gatewayID           string
		natID               string
		publicRouteTableID  string
		privateRouteTableID string
	)

	if err = ensureIAMProfile(cm.conn); err != nil {
		return errors.Wrap(err, "failed to ensure iam profile")
	}

	if err = importPublicKey(cm.conn); err != nil {
		return errors.Wrap(err, "failed to import public key")
	}

	if vpcID, err = ensureVPC(cm.conn); err != nil {
		return errors.Wrap(err, "failed to ensure vpc")
	}

	if publicSubnetID, privateSubnetID, err = ensureSubnet(cm.conn); err != nil {
		return errors.Wrap(err, "failed to ensure subnets")
	}

	if gatewayID, err = ensureInternetGateway(cm.conn, vpcID); err != nil {
		return errors.Wrap(err, "failed to ensure igw")
	}

	if natID, err = ensureNatGateway(cm.conn, vpcID, publicSubnetID); err != nil {
		return errors.Wrap(err, "failed to ensure nat")
	}

	if publicRouteTableID, privateRouteTableID, err = ensureRouteTable(cm.conn, vpcID, gatewayID, natID, publicSubnetID, privateSubnetID); err != nil {
		return errors.Wrap(err, "failed to ensure route table")
	}

	if err = ensureSecurityGroup(cm.conn, vpcID); err != nil {
		return errors.Wrap(err, "failed to ensure security group")
	}

	if err = ensureBastion(cm.conn, publicSubnetID); err != nil {
		return errors.Wrap(err, "failed to ensure bastion")
	}

	if err = ensureLoadBalancer(cm.conn, publicSubnetID); err != nil {
		return errors.Wrap(err, "failed to ensure load balancer")
	}

	clusterSpec, err := clusterapi_aws.ClusterConfigFromProviderSpec(cm.Cluster.Spec.ClusterAPI.Spec.ProviderSpec)
	if err != nil {
		return errors.Wrap(err, "failed to decode provider spec")
	}

	clusterSpec.NetworkSpec = clusterapi_aws.NetworkSpec{
		VPC: clusterapi_aws.VPCSpec{
			ID:                vpcID,
			CidrBlock:         cm.Cluster.Spec.Config.Cloud.AWS.VpcCIDR,
			InternetGatewayID: types.StringP(gatewayID),
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
				RouteTableID:     types.StringP(publicRouteTableID),
				NatGatewayID:     types.StringP(natID),
			},
			{
				ID:               privateSubnetID,
				CidrBlock:        cm.Cluster.Spec.Config.Cloud.AWS.PrivateSubnetCIDR,
				AvailabilityZone: cm.Cluster.Spec.Config.Cloud.Zone,
				IsPublic:         false,
				RouteTableID:     types.StringP(privateRouteTableID),
			},
		},
	}

	rawClusterSpec, err := clusterapi_aws.EncodeClusterSpec(clusterSpec)
	if err != nil {
		return err
	}

	cm.Cluster.Spec.ClusterAPI.Spec.ProviderSpec.Value = rawClusterSpec
	if _, err = store.StoreProvider.Clusters().Update(cm.Cluster); err != nil {
		return err
	}

	return nil
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

func (cm *ClusterManager) EnsureMaster() error {
	var found bool

	leaderMachine, err := cloud.GetLeaderMachine(cm.Cluster)
	if err != nil {
		return errors.Wrap(err, "failed to get leader machine")
	}

	if found, err = cm.conn.getMaster(leaderMachine.Name); err != nil {
		log.Infoln(err)
	}
	if !found {
		log.Info("Creating master instance")
		script, err := cloud.RenderStartupScript(cm, leaderMachine, "", customTemplate)
		if err != nil {
			return err
		}

		var privateSubnetID, vpcID string
		if vpcID, _, err = cm.conn.getVpc(); err != nil {
			log.Infoln(err)
		}

		if privateSubnetID, _, err = cm.conn.getSubnet(vpcID, "private"); err != nil {
			log.Infoln(err)
		}

		masterInstance, err := cm.conn.startMaster(leaderMachine, privateSubnetID, script)
		if err != nil {
			cm.Cluster.Status.Reason = err.Error()
			err = errors.Wrap(err, "")
			return err
		}

		if _, err = store.StoreProvider.Clusters().Update(cm.Cluster); err != nil {
			return err
		}

		// update master machine spec
		spec, err := clusterapi_aws.MachineConfigFromProviderSpec(leaderMachine.Spec.ProviderSpec)
		if err != nil {
			log.Infof("Error decoding provider spec for machine %q", leaderMachine.Name)
			return err
		}

		spec.AMI = clusterapi_aws.AWSResourceReference{
			ID: types.StringP(cm.Cluster.Spec.Config.Cloud.InstanceImage),
		}

		rootDeviceSize, err := cm.conn.getInstanceRootDeviceSize(masterInstance)
		if err != nil {
			return errors.Wrap(err, "failed to get root device size for master instance")
		}

		spec.RootDeviceSize = *rootDeviceSize

		rawSpec, err := clusterapi_aws.EncodeMachineSpec(spec)
		if err != nil {
			return err
		}
		leaderMachine.Spec.ProviderSpec.Value = rawSpec

		// update master machine status
		statusConfig := clusterapi_aws.AWSMachineProviderStatus{
			InstanceID: masterInstance.InstanceId,
		}

		rawStatus, err := clusterapi_aws.EncodeMachineStatus(&statusConfig)
		if err != nil {
			return err
		}
		leaderMachine.Status.ProviderStatus = rawStatus

		// update in pharmer file
		_, err = store.StoreProvider.Machine(cm.Cluster.Name).Update(leaderMachine)
		if err != nil {
			return errors.Wrap(err, "error updating master machine in pharmer storage")
		}
	}

	return nil
}

func ensureIAMProfile(conn *cloudConnector) error {
	var (
		found bool
		err   error
	)

	if found, err = conn.getIAMProfile(); err != nil {
		log.Infoln(err)
	}

	if !found {
		if err := conn.ensureIAMProfile(); err != nil {
			return err
		}
	}

	return nil
}

func importPublicKey(conn *cloudConnector) error {
	var (
		err error
	)
	if _, err := conn.getPublicKey(); err != nil {
		log.Infoln(err)
	}

	if err = conn.importPublicKey(); err != nil {
		return err
	}

	return nil
}

func ensureVPC(conn *cloudConnector) (string, error) {
	var (
		vpcID string
		found bool
		err   error
	)

	if vpcID, found, err = conn.getVpc(); err != nil {
		log.Infoln(err)
	}
	if !found {
		if vpcID, err = conn.setupVpc(); err != nil {
			return "", err
		}
	}

	return vpcID, nil
}

func ensureSubnet(conn *cloudConnector) (string, string, error) {
	var (
		publicSubnetID  string
		privateSubnetID string
		found           bool
		err             error
	)

	vpcID, _, err := conn.getVpc()
	if err != nil {
		return "", "", errors.Wrapf(err, "vpc not found")
	}

	if publicSubnetID, found, err = conn.getSubnet(vpcID, "public"); err != nil {
		log.Infoln(err)
	}
	if !found {
		if publicSubnetID, err = conn.setupPublicSubnet(vpcID); err != nil {
			return "", "", err
		}
	}

	if privateSubnetID, found, err = conn.getSubnet(vpcID, "private"); err != nil {
		log.Infoln(err)
	}
	if !found {
		if privateSubnetID, err = conn.setupPrivateSubnet(vpcID); err != nil {
			return "", "", err
		}
	}
	return publicSubnetID, privateSubnetID, nil
}

func ensureInternetGateway(conn *cloudConnector, vpcID string) (string, error) {
	var (
		found     bool
		err       error
		gatewayID string
	)

	if gatewayID, found, err = conn.getInternetGateway(vpcID); err != nil {
		log.Infoln(err)
	}
	if !found {
		if _, err = conn.setupInternetGateway(vpcID); err != nil {
			return "", err
		}
	}
	return gatewayID, nil
}

func ensureNatGateway(conn *cloudConnector, vpcID, publicSubnetID string) (string, error) {
	var (
		found bool
		err   error
		natID string
	)
	if natID, found, err = conn.getNatGateway(vpcID); err != nil {
		log.Infoln(err)
	}
	if !found {
		if _, err = conn.setupNatGateway(publicSubnetID); err != nil {
			return "", err
		}
	}
	return natID, nil
}

func ensureRouteTable(conn *cloudConnector, vpcID, gatewayID, natID, publicSubnetID, privateSubnetID string) (string, string, error) {
	var (
		found bool
		err   error
	)
	var publicRouteTableID, privateRouteTableID string
	if publicRouteTableID, found, err = conn.getRouteTable("public", vpcID); err != nil {
		log.Infoln(err)
	}
	if !found {
		if publicRouteTableID, err = conn.setupRouteTable("public", vpcID, gatewayID, natID, publicSubnetID, privateSubnetID); err != nil {
			return "", "", err
		}
	}

	if privateRouteTableID, found, err = conn.getRouteTable("private", vpcID); err != nil {
		log.Infoln(err)
	}
	if !found {

		if privateRouteTableID, err = conn.setupRouteTable("private", vpcID, gatewayID, natID, publicSubnetID, privateSubnetID); err != nil {
			return "", "", err
		}
	}
	return publicRouteTableID, privateRouteTableID, nil
}

func ensureSecurityGroup(conn *cloudConnector, vpcID string) error {
	var (
		found bool
		err   error
	)
	if _, found, err = conn.getSecurityGroupID(vpcID, conn.Cluster.Spec.Config.Cloud.AWS.MasterSGName); err != nil {
		log.Infoln(err)
	}
	if !found {

		if err = conn.setupSecurityGroups(vpcID); err != nil {
			conn.Cluster.Status.Reason = err.Error()
			err = errors.Wrap(err, "")
			return err
		}
	} else {
		if err = conn.detectSecurityGroups(vpcID); err != nil {
			return err
		}
	}
	return nil
}

func ensureBastion(conn *cloudConnector, publicSubnetID string) error {
	var (
		found bool
		err   error
	)
	if found, err = conn.getBastion(); err != nil {
		log.Infoln(err)
	}
	if !found {
		if err = conn.setupBastion(publicSubnetID); err != nil {
			conn.Cluster.Status.Reason = err.Error()
			err = errors.Wrap(err, "")
			return err
		}
	}
	return nil
}

func ensureLoadBalancer(conn *cloudConnector, publicSubnetID string) error {
	var (
		found bool
		err   error
	)
	var loadbalancerDNS string
	if loadbalancerDNS, err = conn.getLoadBalancer(); err != nil {
		log.Infoln(err)
	}
	if !found {
		if loadbalancerDNS, err = conn.setupLoadBalancer(publicSubnetID); err != nil {
			conn.Cluster.Status.Reason = err.Error()
			err = errors.Wrap(err, "")
			return err
		}
	}

	if loadbalancerDNS == "" {
		return errors.New("load balancer dns can't be empty")
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
		return errors.Wrap(err, "Error setting controlplane endpoints")
	}
	return nil
}

func (cm *ClusterManager) ApplyDelete() error {
	log.Infoln("deleting cluster")

	if cm.Cluster.Status.Phase == api.ClusterReady {
		cm.Cluster.Status.Phase = api.ClusterDeleting
	}
	if _, err := store.StoreProvider.Clusters().UpdateStatus(cm.Cluster); err != nil {
		return err
	}

	err := cloud.DeleteAllWorkerMachines(cm)
	if err != nil {
		log.Infof("failed to delete nodes: %v", err)
	}

	if err := cm.conn.deleteInstance("controlplane"); err != nil {
		log.Infof("Failed to delete master instance. Reason: %s", err)
	}

	if err := cm.conn.deleteSSHKey(); err != nil {
		log.Infoln(err)
	}

	cm.conn.deleteIAMProfile()

	if _, err := cm.conn.deleteLoadBalancer(); err != nil {
		return errors.Wrap(err, "error deleting load balancer")
	}

	if err := cm.conn.deleteInstance(bastion); err != nil {
		return errors.Wrap(err, "error deleting bastion instance")
	}

	clusterSpec, err := clusterapi_aws.ClusterConfigFromProviderSpec(cm.Cluster.Spec.ClusterAPI.Spec.ProviderSpec)
	if err != nil {
		return err
	}

	vpcID, found, _ := cm.conn.getVpc()
	if !found {
		log.Infof("vpc already deleted")
		return nil
	}

	var natID string
	if len(clusterSpec.NetworkSpec.Subnets) > 0 {
		if clusterSpec.NetworkSpec.Subnets[0].NatGatewayID != nil {
			natID = *clusterSpec.NetworkSpec.Subnets[0].NatGatewayID
		} else {
			natID = *clusterSpec.NetworkSpec.Subnets[0].NatGatewayID
		}
	}

	if err := cm.conn.deleteSecurityGroup(vpcID); err != nil {
		return errors.Wrap(err, "error deleting security group")
	}

	if err := cm.conn.deleteRouteTable(vpcID); err != nil {
		return errors.Wrap(err, "error deleting route table")
	}

	if err := cm.conn.deleteNatGateway(natID); err != nil {
		return errors.Wrap(err, "error deleting NAT")
	}

	if err := cm.conn.releaseReservedIP(); err != nil {
		return errors.Wrap(err, "error deleting Elastic IP")
	}

	if err := cm.conn.deleteInternetGateway(vpcID); err != nil {
		return errors.Wrap(err, "error deleting Interget Gateway")
	}

	if err := cm.conn.deleteSubnetID(vpcID); err != nil {
		return errors.Wrap(err, "error deleting Subnets")
	}

	if err := cm.conn.deleteVpc(vpcID); err != nil {
		return errors.Wrap(err, "error deleting VPC")
	}

	return nil
}
