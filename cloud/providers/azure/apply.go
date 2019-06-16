package azure

import (
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/pubkeypin"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/appscode/go/log"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	capiAzure "github.com/pharmer/pharmer/apis/v1beta1/azure"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
)

func (cm *ClusterManager) PrepareCloud() error {
	err := ensureResourceGroup(cm.conn)
	if err != nil {
		return errors.Wrapf(err, "failed to ensure resource group")
	}

	err = ensureVirtualNetwork(cm.conn)
	if err != nil {
		return errors.Wrapf(err, "failed to ensure virtual network")
	}

	controlPlaneSG, nodeSG, err := ensureSecurityGroup(cm.conn)
	if err != nil {
		return errors.Wrapf(err, "failed to ensure security group")
	}

	routeTable, err := ensureRouteTable(cm.conn)
	if err != nil {
		return errors.Wrapf(err, "failed to ensure route table")
	}

	controlPlaneSubnet, err := ensureSubnet(cm.conn, controlPlaneSG, nodeSG, routeTable)
	if err != nil {
		return errors.Wrapf(err, "failed to ensure subnets")
	}

	publicLB, internalLB, err := ensureLoadBalancer(cm.conn, controlPlaneSubnet)
	if err != nil {
		return errors.Wrapf(err, "failed to ensure load balancer")
	}

	err = ensureMasterNIC(cm.conn, cm.Cluster.MasterMachineName(0), controlPlaneSubnet, publicLB, internalLB)

	return nil
}

func ensureResourceGroup(conn *cloudConnector) error {
	found, err := conn.getResourceGroup()
	if err != nil {
		log.Infoln(err)
	}
	if !found {
		if _, err = conn.ensureResourceGroup(); err != nil {
			return err
		}
	}

	return nil
}

func ensureVirtualNetwork(conn *cloudConnector) error {
	if _, err := conn.getVirtualNetwork(); err != nil {
		if _, err = conn.ensureVirtualNetwork(); err != nil {
			return err
		}
	}

	return nil
}

func ensureSecurityGroup(conn *cloudConnector) (*network.SecurityGroup, *network.SecurityGroup, error) {
	var controlplaneSG network.SecurityGroup
	controlplaneSG, err := conn.getNetworkSecurityGroup(conn.namer.GenerateControlPlaneSecurityGroupName())
	if err != nil {
		if controlplaneSG, err = conn.createNetworkSecurityGroup(true); err != nil {
			return nil, nil, err
		}
	}

	var nodeSG network.SecurityGroup
	if nodeSG, err = conn.getNetworkSecurityGroup(conn.namer.GenerateNodeSecurityGroupName()); err != nil {
		if nodeSG, err = conn.createNetworkSecurityGroup(false); err != nil {
			return nil, nil, err
		}
	}

	return &controlplaneSG, &nodeSG, nil
}

func ensureRouteTable(conn *cloudConnector) (*network.RouteTable, error) {
	rt, err := conn.getRouteTable()
	if err != nil {
		if rt, err = conn.createRouteTable(); err != nil {
			return nil, err
		}
	}
	return &rt, err
}

func ensureSubnet(conn *cloudConnector, controlplaneSG, nodeSG *network.SecurityGroup, rt *network.RouteTable) (*network.Subnet, error) {
	controlPlaneSN, err := conn.getSubnetID(conn.namer.GenerateControlPlaneSubnetName())
	if err != nil {
		controlPlaneSN, err = conn.createSubnetID(conn.namer.GenerateControlPlaneSubnetName(), controlplaneSG, nil)
		if err != nil {
			return nil, err
		}
	}

	_, err = conn.getSubnetID(conn.namer.GenerateNodeSubnetName())
	if err != nil {
		if _, err = conn.createSubnetID(conn.namer.GenerateNodeSubnetName(), nodeSG, rt); err != nil {
			return nil, err
		}
	}

	return &controlPlaneSN, nil
}

func ensureLoadBalancer(conn *cloudConnector, controlPlaneSN *network.Subnet) (*network.LoadBalancer, *network.LoadBalancer, error) {
	internalLB, err := conn.findLoadBalancer(conn.namer.GenerateInternalLBName())
	if err != nil {
		internalLB, err = conn.createInternalLoadBalancer(conn.namer.GenerateInternalLBName(), controlPlaneSN)
		if err != nil {
			return nil, nil, err
		}
	}

	lbPIP, err := conn.getPublicIP(conn.namer.GeneratePublicIPName())
	if err != nil {
		if lbPIP, err = conn.createPublicIP(conn.namer.GeneratePublicIPName()); err != nil {
			conn.Cluster.Status.Reason = err.Error()
			return nil, nil, err
		}
	}

	publicLB, err := conn.findLoadBalancer(conn.namer.GeneratePublicLBName())
	if err != nil {
		if publicLB, err = conn.createPublicLoadBalancer(&lbPIP); err != nil {
			return nil, nil, err
		}
	}

	// update load balancer field
	conn.Cluster.Status.Cloud.LoadBalancer = api.LoadBalancer{
		DNS:  *lbPIP.DNSSettings.Fqdn,
		IP:   *lbPIP.IPAddress,
		Port: api.DefaultKubernetesBindPort,
	}

	nodeAddresses := []core.NodeAddress{
		{
			Type:    core.NodeExternalDNS,
			Address: conn.Cluster.Status.Cloud.LoadBalancer.DNS,
		},
	}

	if err = conn.Cluster.SetClusterApiEndpoints(nodeAddresses); err != nil {
		return nil, nil, err
	}

	return &publicLB, &internalLB, nil
}

func ensureMasterNIC(conn *cloudConnector, machineName string, controlPlaneSN *network.Subnet, publicLB, internalLB *network.LoadBalancer) error {
	_, err := conn.getNetworkInterface(conn.namer.NetworkInterfaceName(machineName))
	if err != nil {
		_, err = conn.createNetworkInterface(conn.namer.NetworkInterfaceName(machineName), controlPlaneSN, publicLB, internalLB)
		if err != nil {
			return err
		}
	}
	return nil
}

func (cm *ClusterManager) GetMasterSKU(totalNodes int32) string {
	return "Standard_B2ms"
}

func (cm *ClusterManager) EnsureMaster() error {
	leaderMachine, err := GetLeaderMachine(cm.Cluster)
	if err != nil {
		return err
	}

	masterNIC, err := cm.conn.getNetworkInterface(cm.namer.NetworkInterfaceName(leaderMachine.Name))
	if err != nil {
		return errors.Wrapf(err, "failed to get master nic")
	}

	_, err = cm.conn.getVirtualMachine(leaderMachine.Name)

	if err != nil {
		script, err := RenderStartupScript(cm, leaderMachine, "", customTemplate)
		if err != nil {
			return err
		}

		vm, err := cm.conn.createVirtualMachine(masterNIC.ID, leaderMachine.Name, script, leaderMachine)
		if err != nil {
			return err
		}

		if _, err = store.StoreProvider.Clusters().Update(cm.Cluster); err != nil {
			return err
		}

		// update master machine status
		statusConfig := capiAzure.AzureMachineProviderStatus{
			VMID: vm.ID,
		}

		rawStatus, err := capiAzure.EncodeMachineStatus(&statusConfig)
		if err != nil {
			return err
		}
		leaderMachine.Status.ProviderStatus = rawStatus

		// update in pharmer file
		leaderMachine, err = store.StoreProvider.Machine(cm.Cluster.Name).Update(leaderMachine)
		if err != nil {
			return errors.Wrap(err, "error updating master machine in pharmer storage")
		}
	}

	kubeConfig, err := GetAdminConfig(cm)
	if err != nil {
		return err
	}

	config := api.Convert_KubeConfig_To_Config(kubeConfig)
	data, err := clientcmd.Write(*config)
	if err != nil {
		return err
	}

	clusterConfig, err := capiAzure.ClusterConfigFromProviderSpec(cm.Cluster.Spec.ClusterAPI.Spec.ProviderSpec)
	if err != nil {
		return err
	}
	clusterConfig.AdminKubeconfig = string(data)
	clusterConfig.DiscoveryHashes = append(clusterConfig.DiscoveryHashes, pubkeypin.Hash(cm.Certs.CACert.Cert))

	rawConfig, err := capiAzure.EncodeClusterSpec(clusterConfig)
	if err != nil {
		return err
	}
	cm.Cluster.Spec.ClusterAPI.Spec.ProviderSpec.Value = rawConfig

	return nil
}

func (cm *ClusterManager) ApplyDelete() error {
	if err := cm.conn.deleteResourceGroup(); err != nil {
		return err
	}
	// Failed
	cm.Cluster.Status.Phase = api.ClusterDeleted
	_, err := store.StoreProvider.Clusters().UpdateStatus(cm.Cluster)
	if err != nil {
		return err
	}

	return nil
}
