package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	. "github.com/appscode/go/types"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud/cmds/options"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/util/cert"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

var managedProviders = sets.NewString("aks", "gke", "eks")

func List(ctx context.Context, opts metav1.ListOptions) ([]*api.Cluster, error) {
	return Store(ctx).Clusters().List(opts)
}

func Get(ctx context.Context, name string) (*api.Cluster, error) {
	return Store(ctx).Clusters().Get(name)
}

func Create(ctx context.Context, cluster *api.Cluster) (*api.Cluster, error) {
	config := cluster.Spec.Config
	if cluster == nil {
		return nil, errors.New("missing cluster")
	} else if cluster.Name == "" {
		return nil, errors.New("missing cluster name")
	} else if config.KubernetesVersion == "" {
		return nil, errors.New("missing cluster version")
	}

	_, err := Store(ctx).Clusters().Get(cluster.Name)
	if err == nil {
		return nil, errors.Errorf("cluster exists with name `%s`", cluster.Name)
	}

	cm, err := GetCloudManager(config.Cloud.CloudProvider, ctx)
	if err != nil {
		return nil, err
	}

	if err = cm.SetDefaultCluster(cluster, config); err != nil {
		return nil, err
	}
	if cluster, err = Store(ctx).Clusters().Create(cluster); err != nil {
		return nil, err
	}

	if ctx, err = CreateCACertificates(ctx, cluster); err != nil {
		return nil, err
	}
	/*if ctx, err = CreateApiserverCertificates(ctx, cluster); err != nil {
		return nil, err
	}
	if ctx, err = CreateServiceAccountKey(ctx, cluster); err != nil {
		return nil, err
	}
	if ctx, err = CreateEtcdCertificates(ctx, cluster); err != nil {
		return nil, err
	}*/
	if ctx, err = CreateSSHKey(ctx, cluster); err != nil {
		return nil, err
	}
	if !managedProviders.Has(cluster.ClusterConfig().Cloud.CloudProvider) {
		master, err := CreateMasterMachines(ctx, cluster)
		if err != nil {
			return nil, err
		}
		if _, err = Store(ctx).Machine(cluster.Name).Create(master); err != nil {
			return nil, err
		}
	}

	return cluster, nil
}

func CreateMasterMachines(ctx context.Context, cluster *api.Cluster) (*clusterapi.Machine, error) {
	cm, err := GetCloudManager(cluster.ClusterConfig().Cloud.CloudProvider, ctx)
	if err != nil {
		return nil, err
	}
	providerSpec, err := cm.GetDefaultProviderSpec(cluster, "")
	if err != nil {
		return nil, err
	}

	/*role := api.RoleMember
	if ind == 0 {
		role = api.RoleLeader
	}*/
	machine := &clusterapi.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:              fmt.Sprintf("%v-master", cluster.Name),
			UID:               uuid.NewUUID(),
			CreationTimestamp: metav1.Time{Time: time.Now()},
			Labels: map[string]string{
				"set":             "master",
				api.RoleMasterKey: "",
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
	api.AssignTypeKind(machine)

	return machine, nil
}

func CreateNodeGroup(ctx context.Context, cluster *api.Cluster, sku string, nodeType api.NodeType, count int32, spotPriceMax float64) error {
	cm, err := GetCloudManager(cluster.ClusterConfig().Cloud.CloudProvider, ctx)
	if err != nil {
		return err
	}

	spec, err := cm.GetDefaultNodeSpec(cluster, sku)
	if err != nil {
		return err
	}

	spec.Type = nodeType
	if nodeType == api.NodeTypeSpot {
		spec.SpotPriceMax = spotPriceMax
	}
	nodeConf := api.MachineProviderConfig{
		Name:   "",
		Config: spec,
	}
	providerConfValue, err := json.Marshal(nodeConf)
	if err != nil {
		return err
	}

	ig := clusterapi.MachineSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:              strings.Replace(sku, "_", "-", -1) + "-pool",
			CreationTimestamp: metav1.Time{Time: time.Now()},
		},
		Spec: clusterapi.MachineSetSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					api.PharmerCluster:  cluster.Name,
					api.MachineSlecetor: sku,
				},
			},
			Replicas: Int32P(count),
			Template: clusterapi.MachineTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						api.PharmerCluster:  cluster.Name,
						api.RoleNodeKey:     "",
						api.MachineSlecetor: sku,
					},
					CreationTimestamp: metav1.Time{Time: time.Now()},
				},
				Spec: clusterapi.MachineSpec{
					ProviderSpec: clusterapi.ProviderSpec{
						Value: &runtime.RawExtension{
							Raw: providerConfValue,
						},
					},
					Versions: clusterapi.MachineVersionInfo{
						ControlPlane: cluster.ClusterConfig().KubernetesVersion,
					},
				},
			},
		},
	}

	_, err = Store(ctx).MachineSet(cluster.Name).Create(&ig)

	return err
}

func Delete(ctx context.Context, name string) (*api.Cluster, error) {
	if name == "" {
		return nil, errors.New("missing cluster name")
	}

	cluster, err := Store(ctx).Clusters().Get(name)
	if err != nil {
		return nil, errors.Errorf("cluster `%s` does not exist. Reason: %v", name, err)
	}
	cluster.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	cluster.Status.Phase = api.ClusterDeleting

	return Store(ctx).Clusters().Update(cluster)
}

func DeleteNG(ctx context.Context, clusterName, nodeGroupName string) error {
	if clusterName == "" {
		return errors.New("missing cluster name")
	}
	if nodeGroupName == "" {
		return errors.New("missing nodegroup name")
	}

	if _, err := Store(ctx).Clusters().Get(clusterName); err != nil {
		return errors.Errorf("cluster `%s` does not exist. Reason: %v", clusterName, err)
	}

	nodeGroup, err := Store(ctx).NodeGroups(clusterName).Get(nodeGroupName)
	if err != nil {
		return errors.Errorf(`nodegroup not found`)
	}

	//	if !nodeGroup.IsMaster() {
	//		nodeGroup.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	_, err = Store(ctx).NodeGroups(clusterName).Update(nodeGroup)
	return err
	//	}

	return nil
}

func DeleteMachineSet(ctx context.Context, clusterName, setName string) error {
	if clusterName == "" {
		return errors.New("missing cluster name")
	}
	if setName == "" {
		return errors.New("missing machineset name")
	}

	mSet, err := Store(ctx).MachineSet(clusterName).Get(setName)
	if err != nil {
		return errors.Errorf(`machinset not found in pharmer db, try using kubectl`)
	}
	mSet.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	_, err = Store(ctx).MachineSet(clusterName).Update(mSet)
	return err
}

func GetSSHConfig(ctx context.Context, nodeName string, cluster *api.Cluster) (*api.SSHConfig, error) {
	var err error
	ctx, err = LoadCACertificates(ctx, cluster)
	if err != nil {
		return nil, err
	}
	client, err := NewAdminClient(ctx, cluster)
	if err != nil {
		return nil, err
	}
	node, err := client.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	ctx, err = LoadSSHKey(ctx, cluster)
	if err != nil {
		return nil, err
	}

	cm, err := GetCloudManager(cluster.ClusterConfig().Cloud.CloudProvider, ctx)
	if err != nil {
		return nil, err
	}
	return cm.GetSSHConfig(cluster, node)
}

func GetAdminConfig(ctx context.Context, cluster *api.Cluster) (*api.KubeConfig, error) {
	if managedProviders.Has(cluster.ClusterConfig().Cloud.CloudProvider) {
		cm, err := GetCloudManager(cluster.ClusterConfig().Cloud.CloudProvider, ctx)
		if err != nil {
			return nil, err
		}
		return cm.GetKubeConfig(cluster)
	}
	var err error
	ctx, err = LoadCACertificates(ctx, cluster)
	if err != nil {
		return nil, err
	}
	adminCert, adminKey, err := CreateAdminCertificate(ctx)
	if err != nil {
		return nil, err
	}

	var (
		clusterName = fmt.Sprintf("%s.pharmer", cluster.Name)
		userName    = fmt.Sprintf("cluster-admin@%s.pharmer", cluster.Name)
		ctxName     = fmt.Sprintf("cluster-admin@%s.pharmer", cluster.Name)
	)
	cfg := api.KubeConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "KubeConfig",
		},
		Preferences: api.Preferences{
			Colors: true,
		},
		Cluster: api.NamedCluster{
			Name:                     clusterName,
			Server:                   cluster.APIServerURL(),
			CertificateAuthorityData: cert.EncodeCertPEM(CACert(ctx)),
		},
		AuthInfo: api.NamedAuthInfo{
			Name:                  userName,
			ClientCertificateData: cert.EncodeCertPEM(adminCert),
			ClientKeyData:         cert.EncodePrivateKeyPEM(adminKey),
		},
		Context: api.NamedContext{
			Name:     ctxName,
			Cluster:  clusterName,
			AuthInfo: userName,
		},
	}
	return &cfg, nil
}

func Apply(ctx context.Context, opts *options.ApplyConfig) ([]api.Action, error) {
	if opts.ClusterName == "" {
		return nil, errors.New("missing cluster name")
	}

	cluster, err := Store(ctx).Clusters().Get(opts.ClusterName)
	if err != nil {
		return nil, errors.Errorf("cluster `%s` does not exist. Reason: %v", opts.ClusterName, err)
	}

	cm, err := GetCloudManager(cluster.ClusterConfig().Cloud.CloudProvider, ctx)
	if err != nil {
		return nil, err
	}

	return cm.Apply(cluster, opts.DryRun)
}

func CheckForUpdates(ctx context.Context, name string) (string, error) {
	if name == "" {
		return "", errors.New("missing cluster name")
	}

	cluster, err := Store(ctx).Clusters().Get(name)
	if err != nil {
		return "", errors.Errorf("cluster `%s` does not exist. Reason: %v", name, err)
	}
	if cluster.Status.Phase == "" {
		return "", errors.Errorf("cluster `%s` is in unknown phase", cluster.Name)
	}
	if cluster.Status.Phase != api.ClusterReady {
		return "", errors.Errorf("cluster `%s` is not ready", cluster.Name)
	}
	if cluster.Status.Phase == api.ClusterDeleted {
		return "", nil
	}
	if ctx, err = LoadCACertificates(ctx, cluster); err != nil {
		return "", err
	}
	if ctx, err = LoadSSHKey(ctx, cluster); err != nil {
		return "", err
	}
	kc, err := NewAdminClient(ctx, cluster)
	if err != nil {
		return "", err
	}
	cm, err := GetCloudManager(cluster.ClusterConfig().Cloud.CloudProvider, ctx)
	if err != nil {
		return "", err
	}
	upm := NewUpgradeManager(ctx, cm, kc, cluster)
	upgrades, err := upm.GetAvailableUpgrades()
	if err != nil {
		return "", err
	}
	upm.PrintAvailableUpgrades(upgrades)
	return "", nil
}

func UpdateSpec(ctx context.Context, cluster *api.Cluster) (*api.Cluster, error) {
	if cluster == nil {
		return nil, errors.New("missing cluster")
	} else if cluster.Name == "" {
		return nil, errors.New("missing cluster name")
	} else if cluster.ClusterConfig().KubernetesVersion == "" {
		return nil, errors.New("missing cluster version")
	}

	existing, err := Store(ctx).Clusters().Get(cluster.Name)
	if err != nil {
		return nil, errors.Errorf("cluster `%s` does not exist. Reason: %v", cluster.Name, err)
	}
	cluster.Status = existing.Status
	cluster.Generation = time.Now().UnixNano()

	return Store(ctx).Clusters().Update(cluster)
}
