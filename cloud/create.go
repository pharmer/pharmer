package cloud

import (
	"strings"
	"time"

	"github.com/appscode/go/types"
	"github.com/google/uuid"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud/cmds/options"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	api_types "k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/apis/core"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func Create(store store.ResourceInterface, cluster *api.Cluster) (Interface, error) {
	if cluster == nil {
		return nil, errors.New("missing Cluster")
	} else if cluster.Name == "" {
		return nil, errors.New("missing Cluster name")
	} else if cluster.Spec.Config.KubernetesVersion == "" {
		return nil, errors.New("missing Cluster version")
	}

	// create should return error: Cluster already exists if Cluster already exists
	_, err := store.Clusters().Get(cluster.Name)
	if err == nil {
		return nil, errors.New("Cluster already exists")
	}

	// create certificates
	certs, err := createPharmerCerts(store, cluster)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create certificates")
	}

	cm, err := GetCloudManagerWithCerts(cluster, certs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cloud manager")
	}

	// create cluster
	if err := createPharmerCluster(store, cm); err != nil {
		return nil, errors.Wrap(err, "failed to create certificates")
	}

	// create master machines
	if err := createMasterMachines(store, cm); err != nil {
		return nil, errors.Wrap(err, "failed to create master machines")
	}

	return cm, nil
}

func createPharmerCluster(store store.ResourceInterface, cm Interface) error {
	cluster := cm.GetCluster()

	// set common cluster configs
	err := setDefaultCluster(cluster)
	if err != nil {
		return errors.Wrap(err, "failed to set default Cluster")
	}

	// set cloud-specific configs
	if err = cm.SetDefaultCluster(); err != nil {
		return errors.Wrap(err, "failed to set provider defaults")
	}

	_, err = store.Clusters().Create(cluster)
	if err != nil {
		return errors.Wrap(err, "failed to store cluster")
	}

	return nil
}

func createMasterMachines(store store.ResourceInterface, cm Interface) error {
	cluster := cm.GetCluster()
	if !managedProviders.Has(cluster.ClusterConfig().Cloud.CloudProvider) {
		for i := 0; i < cluster.Spec.Config.MasterCount; i++ {
			master, err := getMasterMachine(cm, cluster.MasterMachineName(i))
			if err != nil {
				return errors.Wrap(err, "failed to create master machine")
			}
			if _, err = store.Machine(cluster.Name).Create(master); err != nil {
				return errors.Wrap(err, "failed to store mastet machine")
			}
		}
	}
	return nil
}

func setDefaultCluster(cluster *api.Cluster) error {
	config := cluster.Spec.Config

	uid, err := uuid.NewUUID()
	if err != nil {
		return errors.Wrap(err, "failed to create cluster uuid")
	}

	cluster.ObjectMeta.UID = api_types.UID(uid.String())
	cluster.ObjectMeta.CreationTimestamp = metav1.Time{Time: time.Now()}
	cluster.ObjectMeta.Generation = time.Now().UnixNano()

	if err := api.AssignTypeKind(cluster); err != nil {
		return errors.Wrap(err, "failed to assign apiversion and kind to cluster")
	}
	if err := api.AssignTypeKind(&cluster.Spec.ClusterAPI); err != nil {
		return errors.Wrap(err, "failed to assign apiversion and kind to clusterAPI object")
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

func getMasterMachine(cm Interface, name string) (*clusterapi.Machine, error) {
	cluster := cm.GetCluster()
	providerSpec, err := cm.GetDefaultMachineProviderSpec("", api.MasterMachineRole)
	if err != nil {
		return nil, err
	}

	machine := &clusterapi.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
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
		return nil, errors.Wrap(err, "failed to assign apiversion/kind to machine")
	}

	return machine, nil
}

func CreateMachineSets(store store.ResourceInterface, cm Interface, opts *options.NodeGroupCreateConfig) error {
	for sku, count := range opts.Nodes {
		machineset, err := getMachineSet(cm, sku, int32(count))
		if err != nil {
			return errors.Wrap(err, "failed to create machineset")
		}
		_, err = store.MachineSet(cm.GetCluster().Name).Create(machineset)
		if err != nil {
			return errors.Wrap(err, "failed to store machineset")
		}
	}
	return nil
}

func getMachineSet(cm Interface, sku string, count int32) (*clusterapi.MachineSet, error) {
	cluster := cm.GetCluster()

	providerSpec, err := cm.GetDefaultMachineProviderSpec(sku, api.NodeMachineRole)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get default machine provider spec")
	}

	machineSet := &clusterapi.MachineSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:              GenerateMachineSetName(sku),
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

	return machineSet, nil

}

func GenerateMachineSetName(sku string) string {
	return strings.Replace(strings.ToLower(sku), "_", "-", -1) + "-pool"
}
