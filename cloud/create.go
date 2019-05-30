package cloud

import (
	"fmt"
	"strings"
	"time"

	"github.com/pharmer/pharmer/cloud/cmds/options"

	"github.com/google/uuid"

	"github.com/appscode/go/types"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	api_types "k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/apis/core"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func Create(cluster *api.Cluster) (Interface, *api.Cluster, error) {
	config := cluster.Spec.Config
	if cluster == nil {
		return nil, nil, errors.New("missing Cluster")
	} else if cluster.Name == "" {
		return nil, nil, errors.New("missing Cluster name")
	} else if config.KubernetesVersion == "" {
		return nil, nil, errors.New("missing Cluster version")
	}

	// create should return error=Cluster already exists if Cluster already exists
	_, err := store.StoreProvider.Clusters().Get(cluster.Name)
	if err == nil {
		return nil, nil, errors.New("Cluster already exists")
	}

	// set common Cluster configs
	err = SetDefaultCluster(cluster)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to set default Cluster")
	}

	certs, err := createPharmerCerts(cluster)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create certificates")
	}

	cm, err := GetCloudManagerWithCerts(cluster, certs)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get cloud manager")
	}

	// set cloud-specific configs
	if err = cm.SetDefaultCluster(cluster); err != nil {
		return nil, nil, errors.Wrap(err, "failed to set default Cluster")
	}

	if !managedProviders.Has(cluster.ClusterConfig().Cloud.CloudProvider) {
		for i := 0; i < cluster.Spec.Config.MasterCount; i++ {
			master, err := CreateMasterMachines(cm, i)
			if err != nil {
				return nil, nil, err
			}
			if _, err = store.StoreProvider.Machine(cluster.Name).Create(master); err != nil {
				return nil, nil, err
			}
		}
	}

	cluster, err = store.StoreProvider.Clusters().Create(cluster)
	if err != nil {
		return nil, nil, err
	}

	return cm, cluster, nil
}

func SetDefaultCluster(cluster *api.Cluster) error {
	config := cluster.Spec.Config

	uid, _ := uuid.NewUUID()
	cluster.ObjectMeta.UID = api_types.UID(uid.String())
	cluster.ObjectMeta.CreationTimestamp = metav1.Time{Time: time.Now()}
	cluster.ObjectMeta.Generation = time.Now().UnixNano()

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
	cluster.Status = api.PharmerClusterStatus{
		Phase: api.ClusterPending,
	}

	return nil
}

func CreateMasterMachines(cm Interface, index int) (*clusterapi.Machine, error) {
	cluster := cm.GetCluster()
	providerSpec, err := cm.GetDefaultMachineProviderSpec(cluster, "", api.MasterRole)
	if err != nil {
		return nil, err
	}

	machine := &clusterapi.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:              fmt.Sprintf("%v-master-%v", cluster.Name, index),
			CreationTimestamp: metav1.Time{Time: time.Now()},
			Labels: map[string]string{
				"set":                              "controlplane",
				api.RoleMasterKey:                  "",
				clusterapi.MachineClusterLabelName: cluster.Name,
			},
		},
		Spec: clusterapi.MachineSpec{
			ProviderSpec: providerSpec,
			Versions: clusterapi.MachineVersionInfo{
				Kubelet:      cluster.ClusterConfig().KubernetesVersion,
				ControlPlane: cluster.ClusterConfig().KubernetesVersion,
			},
		},
	}
	if err := api.AssignTypeKind(machine); err != nil {
		return nil, err
	}

	return machine, nil
}

func CreateMachineSet(cm Interface, sku string, count int32) error {
	cluster := cm.GetCluster()

	providerSpec, err := cm.GetDefaultMachineProviderSpec(cluster, sku, api.RoleNode)
	if err != nil {
		return errors.Wrap(err, "failed to get default machine provider spec")
	}

	machineSet := clusterapi.MachineSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:              strings.Replace(strings.ToLower(sku), "_", "-", -1) + "-pool",
			CreationTimestamp: metav1.Time{Time: time.Now()},
		},
		Spec: clusterapi.MachineSetSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					clusterapi.MachineClusterLabelName: cluster.Name,
					api.MachineSlecetor:                sku,
				},
			},
			Replicas: types.Int32P(count),
			Template: clusterapi.MachineTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						api.PharmerCluster:                 cluster.Name,
						api.RoleNodeKey:                    "",
						api.MachineSlecetor:                sku,
						"set":                              "node",
						clusterapi.MachineClusterLabelName: cluster.Name, //ref:https://github.com/kubernetes-sigs/cluster-api/blob/master/pkg/controller/machine/controller.go#L229-L232
					},
					CreationTimestamp: metav1.Time{Time: time.Now()},
				},
				Spec: clusterapi.MachineSpec{
					ProviderSpec: providerSpec,
					Versions: clusterapi.MachineVersionInfo{
						Kubelet: cluster.ClusterConfig().KubernetesVersion,
					},
				},
			},
		},
	}

	_, err = store.StoreProvider.MachineSet(cluster.Name).Create(&machineSet)

	return err
}

func CreateMachineSetsFromOptions(cm Interface, opts *options.NodeGroupCreateConfig) error {
	for sku, count := range opts.Nodes {
		err := CreateMachineSet(cm, sku, int32(count))
		return err
	}
	return nil
}
