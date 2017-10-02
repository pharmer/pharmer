package azure

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/appscode/go/errors"
	"github.com/appscode/pharmer/api"
	. "github.com/appscode/pharmer/cloud"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AzureNodeGroupManager struct {
	cm       *ClusterManager
	instance Instance
	im       *instanceManager
}

func (igm *AzureNodeGroupManager) AdjustNodeGroup(dryRun bool) (acts []api.Action, err error) {
	acts = make([]api.Action, 0)
	instanceGroupName := igm.cm.namer.GetNodeGroupName(igm.instance.Type.Sku) //igm.cm.ctx.Name + "-" + strings.Replace(igm.instance.Type.Sku, "_", "-", -1) + "-node"
	adjust, _ := Mutator(igm.cm.ctx, igm.cm.cluster, igm.instance, instanceGroupName)
	fmt.Println(adjust)

	message := ""
	var action api.ActionType
	if adjust == 0 {
		message = "No changed will be applied"
		action = api.ActionNOP
	} else if adjust < 0 {
		message = fmt.Sprintf("%v node will be deleted from %v group", -adjust, instanceGroupName)
		action = api.ActionDelete
	} else {
		message = fmt.Sprintf("%v node will be added to %v group", adjust, instanceGroupName)
		action = api.ActionAdd
	}
	acts = append(acts, api.Action{
		Action:   action,
		Resource: "Node",
		Message:  message,
	})
	if adjust == 0 || dryRun {
		return
	}

	igm.cm.cluster.Generation = igm.instance.Type.ContextVersion

	if adjust == igm.instance.Stats.Count {
		err = igm.createNodeGroup(igm.instance.Stats.Count)
	} else if igm.instance.Stats.Count == 0 {
		err = igm.deleteNodeGroup(igm.instance.Type.Sku)
		if err != nil {
			igm.cm.cluster.Status.Reason = err.Error()
			err = errors.FromErr(err).WithContext(igm.cm.ctx).Err()
			return
		}
	} else {
		err = igm.updateNodeGroup(igm.instance.Type.Sku, adjust)
		if err != nil {
			igm.cm.cluster.Status.Reason = err.Error()
			return
		}
	}
	return
}

func (igm *AzureNodeGroupManager) updateNodeGroup(sku string, count int64) error {
	found, instances, err := igm.im.GetNodeGroup(igm.cm.namer.GetNodeGroupName(sku))
	if err != nil {
		return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	if !found {
		return errors.New("Instance group not found").Err()
	}
	if count < 0 {
		for _, instance := range instances {
			err = igm.im.DeleteVirtualMachine(instance.Name)
			if err != nil {
				return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
			}
			err = igm.cm.deleteNodeNetworkInterface(igm.cm.namer.NetworkInterfaceName(instance.Name))
			if err != nil {
				return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
			}
			err = igm.cm.deletePublicIp(igm.cm.namer.PublicIPName(instance.Name))
			if err != nil {
				return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
			}
			count++
			if count >= 0 {
				return nil
			}
		}
	} else {
		as, err := igm.im.getAvailablitySet()
		if err != nil {
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		}

		vn, err := igm.cm.getVirtualNetwork()
		if err != nil {
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		}

		sn, err := igm.cm.getSubnetID(&vn)
		if err != nil {
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		}

		sg, err := igm.cm.getNetworkSecurityGroup()
		if err != nil {
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		}
		for i := int64(0); i < count; i++ {
			igm.StartNode(as, sg, sn)
		}
	}
	return nil
}

func (igm *AzureNodeGroupManager) listInstances(sku string) ([]*api.Node, error) {
	instances := make([]*api.Node, 0)
	kc, err := igm.cm.GetAdminClient()
	nodes, err := kc.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return instances, errors.FromErr(err).WithContext(igm.cm.ctx).Err()

	}
	for _, n := range nodes.Items {
		nl := api.FromMap(n.GetLabels())
		if nl.GetString(api.NodeLabelKey_SKU) == sku && nl.GetString(api.NodeLabelKey_Role) == "node" {
			nodeVM, err := igm.im.conn.vmClient.Get(igm.cm.namer.ResourceGroupName(), n.Name, compute.InstanceView)
			if err != nil {
				return nil, errors.FromErr(err).WithContext(igm.cm.ctx).Err()
			}
			nodeNIC, err := igm.im.conn.interfacesClient.Get(igm.cm.namer.ResourceGroupName(), igm.cm.namer.NetworkInterfaceName(n.Name), "")
			if err != nil {
				return nil, errors.FromErr(err).WithContext(igm.cm.ctx).Err()
			}
			nodePIP, err := igm.im.conn.publicIPAddressesClient.Get(igm.im.namer.ResourceGroupName(), igm.cm.namer.PublicIPName(n.Name), "")
			if err != nil {
				return nil, errors.FromErr(err).WithContext(igm.cm.ctx).Err()
			}
			instance, err := igm.im.newKubeInstance(nodeVM, nodeNIC, nodePIP)
			if err != nil {
				return nil, errors.FromErr(err).WithContext(igm.cm.ctx).Err()
			}
			instance.Spec.Role = api.RoleNode

			instances = append(instances, instance)
		}
	}
	return instances, nil

}
func (igm *AzureNodeGroupManager) createNodeGroup(count int64) error {
	as, err := igm.im.getAvailablitySet()
	if err != nil {
		return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}

	vn, err := igm.cm.getVirtualNetwork()
	if err != nil {
		return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}

	sn, err := igm.cm.getSubnetID(&vn)
	if err != nil {
		return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}

	sg, err := igm.cm.getNetworkSecurityGroup()
	if err != nil {
		return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}

	for i := int64(0); i < count; i++ {
		_, err := igm.StartNode(as, sg, sn)
		if err != nil {
			igm.cm.cluster.Status.Reason = err.Error()
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()

		}
	}
	return nil
}

func (igm *AzureNodeGroupManager) deleteNodeGroup(sku string) error {
	found, instances, err := igm.im.GetNodeGroup(igm.cm.namer.GetNodeGroupName(sku))
	if err != nil {
		return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	if !found {
		return errors.New("Instance group not found").Err()
	}
	for _, instance := range instances {
		err = igm.im.DeleteVirtualMachine(instance.Name)
		if err != nil {
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		}
		err = igm.cm.deleteNodeNetworkInterface(igm.cm.namer.NetworkInterfaceName(instance.Name))
		if err != nil {
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		}
		err = igm.cm.deletePublicIp(igm.cm.namer.PublicIPName(instance.Name))
		if err != nil {
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		}
	}
	return nil
}

func (igm *AzureNodeGroupManager) StartNode(as compute.AvailabilitySet, sg network.SecurityGroup, sn network.Subnet) (*api.Node, error) {
	ki := &api.Node{}

	nodeName := igm.cm.namer.GenNodeName(igm.instance.Type.Sku)
	nodePIP, err := igm.im.createPublicIP(igm.cm.namer.PublicIPName(nodeName), network.Dynamic)
	if err != nil {
		igm.cm.cluster.Status.Reason = err.Error()
		return ki, errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}

	nodeNIC, err := igm.im.createNetworkInterface(igm.cm.namer.NetworkInterfaceName(nodeName), sg, sn, network.Dynamic, "", nodePIP)
	if err != nil {
		igm.cm.cluster.Status.Reason = err.Error()
		return ki, errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}

	sa, err := igm.im.getStorageAccount()
	if err != nil {
		return ki, errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}

	nodeScript, err := RenderStartupScript(igm.cm.ctx, igm.cm.cluster, api.RoleNode, igm.im.namer.GetNodeGroupName(igm.instance.Type.Sku))
	if err != nil {
		return ki, errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	nodeVM, err := igm.im.createVirtualMachine(nodeNIC, as, sa, nodeName, nodeScript, igm.instance.Type.Sku)
	if err != nil {
		igm.cm.cluster.Status.Reason = err.Error()
		return ki, errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}

	nodePIP, err = igm.im.getPublicIP(igm.cm.namer.PublicIPName(nodeName))
	if err != nil {
		igm.cm.cluster.Status.Reason = err.Error()
		return ki, errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}

	ki, err = igm.im.newKubeInstance(nodeVM, nodeNIC, nodePIP)
	if err != nil {
		igm.cm.cluster.Status.Reason = err.Error()
		return &api.Node{}, errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	ki.Spec.Role = api.RoleNode

	Store(igm.cm.ctx).Instances(igm.cm.cluster.Name).Create(ki)
	return ki, nil
}
