package cloud

import (
	"strings"
	"time"

	"github.com/appscode/go/types"
	"github.com/google/uuid"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud/utils/certificates"
	"github.com/pharmer/pharmer/cmds/cloud/options"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	api_types "k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/apis/core"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func CreateCluster(store store.ResourceInterface, cluster *api.Cluster) error {
	if cluster == nil {
		return errors.New("missing Cluster")
	} else if cluster.Name == "" {
		return errors.New("missing Cluster name")
	} else if cluster.Spec.Config.KubernetesVersion == "" {
		return errors.New("missing Cluster version")
	}

	// create should return error: Cluster already exists if Cluster already exists
	_, err := store.Clusters().Get(cluster.Name)
	if err == nil {
		return errors.New("cluster already exists")
	}

	// create certificates and keys
	certs, err := certificates.CreateCertsKeys(store, cluster.Name)
	if err != nil {
		return errors.Wrap(err, "failed to create certificates")
	}

	scope := NewScope(NewScopeParams{
		Cluster:       cluster,
		Certs:         certs,
		StoreProvider: store,
	})

	// create Cluster
	if err := createCluster(scope); err != nil {
		return errors.Wrap(err, "failed to create certificates")
	}

	// create master machines
	if err := createMasterMachines(scope); err != nil {
		return errors.Wrap(err, "failed to create master machines")
	}

	return nil
}

func createCluster(s *Scope) error {
	cluster := s.Cluster
	cm, err := s.GetCloudManager()
	if err != nil {
		return err
	}

	// set common Cluster configs
	err = setDefaultCluster(cluster)
	if err != nil {
		return errors.Wrap(err, "failed to set default Cluster")
	}

	// set cloud-specific configs
	if err = cm.SetDefaultCluster(); err != nil {
		return errors.Wrap(err, "failed to set provider defaults")
	}

	_, err = s.StoreProvider.Clusters().Create(cluster)
	if err != nil {
		return errors.Wrap(err, "failed to store Cluster")
	}

	return nil
}

func createMasterMachines(s *Scope) error {
	cluster := s.Cluster
	if !api.ManagedProviders.Has(cluster.ClusterConfig().Cloud.CloudProvider) {
		for i := 0; i < cluster.Spec.Config.MasterCount; i++ {
			master, err := getMasterMachine(s, cluster.MasterMachineName(i))
			if err != nil {
				return errors.Wrap(err, "failed to create master machine")
			}
			if _, err = s.StoreProvider.Machine(cluster.Name).Create(master); err != nil {
				return errors.Wrap(err, "failed to store mastet machine")
			}
		}
	}
	return nil
}

func setDefaultCluster(cluster *api.Cluster) error {
	config := &cluster.Spec.Config

	uid, err := uuid.NewUUID()
	if err != nil {
		return errors.Wrap(err, "failed to create Cluster uuid")
	}

	cluster.ObjectMeta.UID = api_types.UID(uid.String())
	cluster.ObjectMeta.CreationTimestamp = metav1.Time{Time: time.Now()}
	cluster.ObjectMeta.Generation = time.Now().UnixNano()

	if err := api.AssignTypeKind(cluster); err != nil {
		return errors.Wrap(err, "failed to assign apiversion and kind to Cluster")
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
	}

	if cluster.Spec.AuditSink {
		cluster.Spec.Config.APIServerExtraArgs["audit-dynamic-configuration"] = "true"
		cluster.Spec.Config.APIServerExtraArgs["feature-gates"] = "DynamicAuditing=true"
		cluster.Spec.Config.APIServerExtraArgs["runtime-config"] = "auditregistration.k8s.io/v1alpha1=true"
	}

	config.KubeletExtraArgs = make(map[string]string)

	config.Cloud.Region = cluster.Spec.Config.Cloud.Zone[0 : len(cluster.Spec.Config.Cloud.Zone)-1]
	config.Cloud.SSHKeyName = cluster.GenSSHKeyExternalID()
	cluster.Status = api.PharmerClusterStatus{
		Phase: api.ClusterPending,
	}

	return nil
}

func getMasterMachine(s *Scope, name string) (*clusterapi.Machine, error) {
	cluster := s.Cluster
	cm, err := s.GetCloudManager()
	if err != nil {
		return nil, err
	}

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

func CreateMachineSets(store store.ResourceInterface, opts *options.NodeGroupCreateConfig) error {
	cluster, err := store.Clusters().Get(opts.ClusterName)
	if err != nil {
		return err
	}

	scope := NewScope(NewScopeParams{
		Cluster:       cluster,
		StoreProvider: store,
	})

	for sku, count := range opts.Nodes {
		machineset, err := getMachineSet(scope, sku, int32(count))
		if err != nil {
			return errors.Wrap(err, "failed to create machineset")
		}
		_, err = store.MachineSet(cluster.Name).Create(machineset)
		if err != nil {
			return errors.Wrap(err, "failed to store machineset")
		}
	}
	return nil
}

func getMachineSet(s *Scope, sku string, count int32) (*clusterapi.MachineSet, error) {
	cluster := s.Cluster
	cm, err := s.GetCloudManager()
	if err != nil {
		return nil, err
	}

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

// TODO: move
func GenerateMachineSetName(sku string) string {
	return strings.Replace(strings.ToLower(sku), "_", "-", -1) + "-pool"
}
