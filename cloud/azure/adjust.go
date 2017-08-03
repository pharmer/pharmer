package azure

import (
	"strings"

	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/appscode/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud/common"
	"github.com/appscode/pharmer/contexts"
	"github.com/appscode/pharmer/system"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type InstanceGroupManager struct {
	cm       *clusterManager
	instance common.Instance
	im       *instanceManager
}

func (igm *InstanceGroupManager) AdjustInstanceGroup() error {
	instanceGroupName := igm.cm.namer.GetInstanceGroupName(igm.instance.Type.Sku) //igm.cm.ctx.Name + "-" + strings.Replace(igm.instance.Type.Sku, "_", "-", -1) + "-node"
	found, err := igm.GetInstanceGroup(instanceGroupName)
	if err != nil {
		igm.cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}

	igm.cm.ctx.ContextVersion = igm.instance.Type.ContextVersion
	igm.cm.ctx.Load()

	if !found {
		err = igm.createInstanceGroup(igm.instance.Stats.Count)
	} else if igm.instance.Stats.Count == 0 {
		nodeAdjust, _ := common.Mutator(igm.cm.ctx, igm.instance)
		if nodeAdjust < 0 {
			nodeAdjust = -nodeAdjust
		}
		err := igm.deleteInstanceGroup(instanceGroupName, nodeAdjust)
		if err != nil {
			igm.cm.ctx.StatusCause = err.Error()
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		}
	} else {
		nodeAdjust, _ := common.Mutator(igm.cm.ctx, igm.instance)
		if nodeAdjust < 0 {
			err := igm.deleteInstanceGroup(instanceGroupName, -nodeAdjust)
			if err != nil {
				igm.cm.ctx.StatusCause = err.Error()
				return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
			}
		} else {
			err := igm.createInstanceGroup(nodeAdjust)
			if err != nil {
				igm.cm.ctx.StatusCause = err.Error()
				return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
			}
		}
	}
	return nil
}

func (igm *InstanceGroupManager) GetInstanceGroup(instanceGroup string) (bool, error) {
	vm, err := igm.cm.conn.vmClient.List(igm.cm.namer.ResourceGroupName())
	if err != nil {
		igm.cm.ctx.StatusCause = err.Error()
		return false, errors.FromErr(err).WithContext(igm.cm.ctx).Err()

	}
	for _, i := range *vm.Value {
		name := strings.Split(*i.ID, "/")
		if strings.HasPrefix(name[len(name)-1], instanceGroup) {
			return true, nil
		}

	}
	return false, nil
	//im.ctx.Logger().Infof("Found virtual machine %v", vm)
}

func (igm *InstanceGroupManager) listInstances(sku string) ([]*contexts.KubernetesInstance, error) {
	instances := make([]*contexts.KubernetesInstance, 0)
	kc, err := igm.cm.ctx.NewKubeClient()
	if err != nil {
		return instances, errors.FromErr(err).WithContext(igm.cm.ctx).Err()

	}
	nodes, err := kc.Client.CoreV1().Nodes().List(metav1.ListOptions{})
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
			instance.Role = system.RoleKubernetesPool

			instances = append(instances, instance)
		}
	}
	return instances, nil

}
func (igm *InstanceGroupManager) createInstanceGroup(count int64) error {
	for i := int64(0); i < count; i++ {
		_, err := igm.StartNode()
		if err != nil {
			igm.cm.ctx.StatusCause = err.Error()
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()

		}
	}
	return nil
}

func (igm *InstanceGroupManager) deleteInstanceGroup(instanceGroup string, count int64) error {
	vm, err := igm.cm.conn.vmClient.List(igm.cm.namer.ResourceGroupName())
	if err != nil {
		igm.cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(igm.cm.ctx).Err()

	}
	for _, i := range *vm.Value {
		name := strings.Split(*i.ID, "/")
		instance := name[len(name)-1]
		if strings.HasPrefix(instance, instanceGroup) {
			count--
			err = igm.im.DeleteVirtualMachine(instance)
			if err != nil {
				return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
			}
			err = igm.cm.deleteNodeNetworkInterface(igm.cm.namer.NetworkInterfaceName(instance))
			if err != nil {
				return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
			}
		}
		if count <= 0 {
			break
		}

	}
	return nil
}

func (igm *InstanceGroupManager) StartNode() (*contexts.KubernetesInstance, error) {
	ki := &contexts.KubernetesInstance{}

	nodeName := igm.cm.namer.GenNodeName(igm.instance.Type.Sku)
	nodePIP, err := igm.im.createPublicIP(igm.cm.namer.PublicIPName(nodeName), network.Dynamic)
	if err != nil {
		igm.cm.ctx.StatusCause = err.Error()
		return ki, errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}

	as, err := igm.im.getAvailablitySet()
	if err != nil {
		return ki, errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}

	vn, err := igm.cm.getVirtualNetwork()
	if err != nil {
		return ki, errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}

	sn, err := igm.cm.getSubnetID(&vn)
	if err != nil {
		return ki, errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}

	nodeNIC, err := igm.im.createNetworkInterface(igm.cm.namer.NetworkInterfaceName(nodeName), sn, network.Dynamic, "", nodePIP)
	if err != nil {
		igm.cm.ctx.StatusCause = err.Error()
		return ki, errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}

	sa, err := igm.im.getStorageAccount()
	if err != nil {
		return ki, errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}

	nodeScript := igm.im.RenderStartupScript(igm.cm.ctx.NewScriptOptions(), igm.instance.Type.Sku, system.RoleKubernetesPool)
	nodeVM, err := igm.im.createVirtualMachine(nodeNIC, as, sa, nodeName, nodeScript, igm.instance.Type.Sku)
	if err != nil {
		igm.cm.ctx.StatusCause = err.Error()
		return ki, errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}

	nodePIP, err = igm.im.getPublicIP(igm.cm.namer.PublicIPName(nodeName))
	if err != nil {
		igm.cm.ctx.StatusCause = err.Error()
		return ki, errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}

	ki, err = igm.im.newKubeInstance(nodeVM, nodeNIC, nodePIP)
	if err != nil {
		igm.cm.ctx.StatusCause = err.Error()
		return &contexts.KubernetesInstance{}, errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	ki.Role = system.RoleKubernetesPool
	igm.cm.ins.Instances = append(igm.cm.ins.Instances, ki)
	return ki, nil
}
