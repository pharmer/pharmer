package cloud

import (
	"strings"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	"k8s.io/kubernetes/pkg/apis/core"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func Create(cluster *api.Cluster) (*api.Cluster, error) {
	config := cluster.Spec.Config
	if cluster == nil {
		return nil, errors.New("missing cluster")
	} else if cluster.Name == "" {
		return nil, errors.New("missing cluster name")
	} else if config.KubernetesVersion == "" {
		return nil, errors.New("missing cluster version")
	}

	// create should return error=cluster already exists if cluster already exists
	_, err := store.StoreProvider.Clusters().Get(cluster.Name)
	if err == nil {
		return nil, errors.New("cluster already exists")
	}

	certs, err := createPharmerCerts(cluster)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create certificates")
	}

	cm, err := GetCloudManager(config.Cloud.CloudProvider, cluster, certs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cloud manager")
	}

	// set common cluster configs
	err = SetDefaultCluster(cluster)
	if err != nil {
		return nil, errors.Wrap(err, "failed to set default cluster")
	}

	// set cloud-specific configs
	if err = cm.SetDefaultCluster(cluster); err != nil {
		return nil, errors.Wrap(err, "failed to set default cluster")
	}

	if !managedProviders.Has(cluster.ClusterConfig().Cloud.CloudProvider) {
		for i := 0; i < cluster.Spec.Config.MasterCount; i++ {
			master, err := CreateMasterMachines(cluster, i)
			if err != nil {
				return nil, err
			}
			if _, err = store.StoreProvider.Machine(cluster.Name).Create(master); err != nil {
				return nil, err
			}
		}
	}

	return store.StoreProvider.Clusters().Create(cluster)
}

func SetDefaultCluster(cluster *api.Cluster) error {
	config := cluster.Spec.Config

	if err := api.AssignTypeKind(cluster); err != nil {
		return err
	}
	if err := api.AssignTypeKind(cluster.Spec.ClusterAPI); err != nil {
		return err
	}

	cluster.SetNetworkingDefaults(config.Cloud.NetworkProvider)

	config.APIServerExtraArgs = map[string]string{
		"kubelet-preferred-address-types": strings.Join([]string{
			string(core.NodeInternalIP),
			string(core.NodeInternalDNS),
			string(core.NodeExternalDNS),
			string(core.NodeExternalIP),
		}, ","),
		"cloud-provider": cluster.Spec.Config.Cloud.CloudProvider,
	}

	cluster.Spec.Config.Cloud.Region = cluster.Spec.Config.Cloud.Zone[0 : len(cluster.Spec.Config.Cloud.Zone)-1]
	cluster.Spec.Config.Cloud.SSHKeyName = cluster.GenSSHKeyExternalID()

	return nil
}

func CreateMasterMachines(cluster *api.Cluster, index int) (*clusterapi.Machine, error) {
	//cm, err := GetCloudManager(cluster.ClusterConfig().Cloud.CloudProvider)
	//if err != nil {
	//	return nil, err
	//}
	//providerSpec, err := cm.GetDefaultMachineProviderSpec(cluster, "", api.MasterRole)
	//if err != nil {
	//	return nil, err
	//}
	//
	///*role := api.RoleMember
	//if ind == 0 {
	//	role = api.RoleLeader
	//}*/
	//machine := &clusterapi.Machine{
	//	ObjectMeta: metav1.ObjectMeta{
	//		Name: fmt.Sprintf("%v-master-%v", cluster.Name, index),
	//		//	UID:               uuid.NewUUID(),
	//		CreationTimestamp: metav1.Time{Time: time.Now()},
	//		Labels: map[string]string{
	//			//ref: https://github.com/kubernetes-sigs/cluster-api-provider-aws/blob/94a3a3abc7b1ebdd88ea89889347f5e644e160cf/pkg/cloud/aws/actuators/machine_scope.go#L90-L93
	//			//ref: https://github.com/kubernetes-sigs/cluster-api-provider-aws/blob/94a3a3abc7b1ebdd88ea89889347f5e644e160cf/pkg/cloud/aws/actuators/machine/actuator.go#L89-L92
	//			"set":                              "controlplane",
	//			api.RoleMasterKey:                  "",
	//			clusterapi.MachineClusterLabelName: cluster.Name,
	//		},
	//	},
	//	Spec: clusterapi.MachineSpec{
	//		ProviderSpec: providerSpec,
	//		Versions: clusterapi.MachineVersionInfo{
	//			Kubelet:      cluster.ClusterConfig().KubernetesVersion,
	//			ControlPlane: cluster.ClusterConfig().KubernetesVersion,
	//		},
	//	},
	//}
	//if err := api.AssignTypeKind(machine); err != nil {
	//	return nil, err
	//}
	//
	//return machine, nil
	return nil, nil
}

func CreateMachineSet(cluster *api.Cluster, owner, role, sku string, nodeType api.NodeType, count int32, spotPriceMax float64) error {
	//var err error
	//cm, err := GetCloudManager(cluster.ClusterConfig().Cloud.CloudProvider)
	//if err != nil {
	//	return err
	//}
	//
	//providerSpec, err := cm.GetDefaultMachineProviderSpec(cluster, sku, api.NodeRole)
	//if err != nil {
	//	return err
	//}
	//
	//machineSet := clusterapi.MachineSet{
	//	ObjectMeta: metav1.ObjectMeta{
	//		Name:              strings.Replace(strings.ToLower(sku), "_", "-", -1) + "-pool",
	//		CreationTimestamp: metav1.Time{Time: time.Now()},
	//	},
	//	Spec: clusterapi.MachineSetSpec{
	//		Selector: metav1.LabelSelector{
	//			MatchLabels: map[string]string{
	//				clusterapi.MachineClusterLabelName: cluster.Name,
	//				api.MachineSlecetor:                sku,
	//			},
	//		},
	//		Replicas: types.Int32P(count),
	//		Template: clusterapi.MachineTemplateSpec{
	//			ObjectMeta: metav1.ObjectMeta{
	//				Labels: map[string]string{
	//					api.PharmerCluster:                 cluster.Name,
	//					api.RoleNodeKey:                    "",
	//					api.MachineSlecetor:                sku,
	//					"set":                              "node",
	//					clusterapi.MachineClusterLabelName: cluster.Name, //ref:https://github.com/kubernetes-sigs/cluster-api/blob/master/pkg/controller/machine/controller.go#L229-L232
	//				},
	//				CreationTimestamp: metav1.Time{Time: time.Now()},
	//			},
	//			Spec: clusterapi.MachineSpec{
	//				ProviderSpec: providerSpec,
	//				Versions: clusterapi.MachineVersionInfo{
	//					Kubelet: cluster.ClusterConfig().KubernetesVersion,
	//				},
	//			},
	//		},
	//	},
	//}
	//
	//_, err = store.StoreProvider.MachineSet(cluster.Name).Create(&machineSet)
	//
	//return err
	return nil
}
