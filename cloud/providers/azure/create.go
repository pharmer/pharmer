package azure

import (
	"encoding/base64"
	"encoding/json"
	"net"
	"strings"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/log"
	"github.com/pharmer/cloud/pkg/credential"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	capiAzure "github.com/pharmer/pharmer/apis/v1beta1/azure"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/cert"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) GetDefaultNodeSpec(cluster *api.Cluster, sku string) (api.NodeSpec, error) {
	if sku == "" {
		sku = "Standard_B2ms"
	}
	return api.NodeSpec{
		SKU: sku,
		//	DiskType:      "",
		//	DiskSize:      100,
	}, nil
}

func (cm *ClusterManager) SetOwner(owner string) {
	cm.owner = owner
}

func (cm *ClusterManager) GetDefaultMachineProviderSpec(cluster *api.Cluster, sku string, role api.MachineRole) (clusterapi.ProviderSpec, error) {
	if sku == "" {
		sku = "Standard_B2ms"
	}
	spec := &capiAzure.AzureMachineProviderSpec{
		TypeMeta: metav1.TypeMeta{
			Kind:       api.AzureProviderMachineKind,
			APIVersion: api.AzureProviderGroupName + "/" + api.AzureProviderApiVersion,
		},
		Roles: []capiAzure.MachineRole{
			capiAzure.MachineRole(role),
		},
		VMSize:   sku,
		Location: cluster.Spec.Config.Cloud.Zone,
		Image: capiAzure.Image{
			Publisher: "Canonical",
			Offer:     "UbuntuServer",
			SKU:       "16.04-LTS",
			Version:   "latest",
		},
		OSDisk: capiAzure.OSDisk{
			OSType: "Linux",
			ManagedDisk: capiAzure.ManagedDisk{
				StorageAccountType: "Premium_LRS",
			},
			DiskSizeGB: 30,
		},
		SSHPublicKey:  base64.StdEncoding.EncodeToString(SSHKey(cm.ctx).PublicKey),
		SSHPrivateKey: base64.StdEncoding.EncodeToString(SSHKey(cm.ctx).PrivateKey),
	}
	providerSpecValue, err := json.Marshal(spec)
	if err != nil {
		return clusterapi.ProviderSpec{}, err
	}

	return clusterapi.ProviderSpec{
		Value: &runtime.RawExtension{
			Raw: providerSpecValue,
		},
	}, nil
}

func (cm *ClusterManager) SetDefaultCluster(cluster *api.Cluster, config *api.ClusterConfig) error {
	cm.cluster = cluster
	n := namer{cluster: cluster}

	if err := api.AssignTypeKind(cluster); err != nil {
		return err
	}
	if err := api.AssignTypeKind(cluster.Spec.ClusterAPI); err != nil {
		return err
	}

	// Init spec
	config.Cloud.Region = config.Cloud.Zone
	config.Cloud.SSHKeyName = n.GenSSHKeyExternalID()

	cluster.SetNetworkingDefaults(config.Cloud.NetworkProvider)
	//config.Cloud..NonMasqueradeCIDR = "10.0.0.0/8"
	//config.API.BindPort = kubeadmapi.DefaultAPIBindPort
	config.APIServerCertSANs = NameGenerator(cm.ctx).ExtraNames(cluster.Name)
	config.APIServerExtraArgs = map[string]string{
		// ref: https://github.com/kubernetes/kubernetes/blob/d595003e0dc1b94455d1367e96e15ff67fc920fa/cmd/kube-apiserver/app/options/options.go#L99
		"kubelet-preferred-address-types": strings.Join([]string{
			string(core.NodeExternalDNS),
			string(core.NodeExternalIP),
			string(core.NodeInternalDNS),
			string(core.NodeInternalIP),
		}, ","),
		"cloud-config":   "/etc/kubernetes/azure.json",
		"cloud-provider": cluster.Spec.Config.Cloud.CloudProvider,
	}
	config.Cloud.CCMCredentialName = cluster.Spec.Config.CredentialName

	cred, err := Store(cm.ctx).Owner(cm.owner).Credentials().Get(cluster.Spec.Config.Cloud.CCMCredentialName)
	if err != nil {
		log.Infof("Error getting credential %q: %v", cluster.Spec.Config.Cloud.CCMCredentialName, err)
		return err
	}
	typed := credential.Azure{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		log.Infof("Invalid credential: %v", err)
		return err
	}

	config.Cloud.Azure = &api.AzureSpec{
		StorageAccountName:     n.GenStorageAccountName(),
		RootPassword:           rand.GeneratePassword(),
		VPCCIDR:                DefaultVnetCIDR,
		ControlPlaneSubnetCIDR: DefaultControlPlaneSubnetCIDR,
		NodeSubnetCIDR:         DefaultNodeSubnetCIDR,
		InternalLBIPAddress:    DefaultInternalLBIPAddress,
		AzureDNSZone:           DefaultAzureDNSZone,
		SubscriptionID:         typed.SubscriptionID(),
		ResourceGroup:          n.ResourceGroupName(),
	}

	// Init status
	cluster.Status = api.PharmerClusterStatus{
		Phase: api.ClusterPending,
		Cloud: api.CloudStatus{},
	}

	return cm.SetAzureCluster()
}

// SetAzureCluster sets up Azure ClusterAPI provider specs
func (cm *ClusterManager) SetAzureCluster() error {
	n := namer{cluster: cm.cluster}

	conf := &capiAzure.AzureClusterProviderSpec{
		TypeMeta: metav1.TypeMeta{
			APIVersion: api.AzureProviderGroupName + "/" + api.AzureProviderApiVersion,
			Kind:       api.AzureProviderClusterKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: cm.cluster.Name,
		},
		ResourceGroup: n.ResourceGroupName(),
		Location:      cm.cluster.Spec.Config.Cloud.Region,
	}

	rawSpec, err := capiAzure.EncodeClusterSpec(conf)
	if err != nil {
		return err
	}

	cm.cluster.Spec.ClusterAPI.Spec.ProviderSpec.Value = rawSpec

	return nil
}

// SetupCerts Loads necessary certs in Cluster Spec
func (cm *ClusterManager) SetupCerts() error {
	conf, err := capiAzure.ClusterConfigFromProviderSpec(cm.cluster.Spec.ClusterAPI.Spec.ProviderSpec)
	if err != nil {
		return err
	}

	conf.CAKeyPair = capiAzure.KeyPair{
		Cert: cert.EncodeCertPEM(CACert(cm.ctx)),
		Key:  cert.EncodePrivateKeyPEM(CAKey(cm.ctx)),
	}
	conf.FrontProxyCAKeyPair = capiAzure.KeyPair{
		Cert: cert.EncodeCertPEM(FrontProxyCACert(cm.ctx)),
		Key:  cert.EncodePrivateKeyPEM(FrontProxyCAKey(cm.ctx)),
	}
	conf.EtcdCAKeyPair = capiAzure.KeyPair{
		Cert: cert.EncodeCertPEM(EtcdCaCert(cm.ctx)),
		Key:  cert.EncodePrivateKeyPEM(EtcdCaKey(cm.ctx)),
	}
	conf.SAKeyPair = capiAzure.KeyPair{
		Cert: cert.EncodeCertPEM(SaCert(cm.ctx)),
		Key:  cert.EncodePrivateKeyPEM(SaKey(cm.ctx)),
	}
	conf.SSHPublicKey = base64.StdEncoding.EncodeToString(SSHKey(cm.ctx).PublicKey)
	conf.SSHPrivateKey = base64.StdEncoding.EncodeToString(SSHKey(cm.ctx).PrivateKey)

	rawSpec, err := capiAzure.EncodeClusterSpec(conf)
	if err != nil {
		return err
	}

	cm.cluster.Spec.ClusterAPI.Spec.ProviderSpec.Value = rawSpec

	if _, err := Store(cm.ctx).Owner(cm.owner).Clusters().Update(cm.cluster); err != nil {
		return err
	}
	return nil
}

func (cm *ClusterManager) SetDefaults(cluster *api.Cluster) error {
	return nil
}

func (cm *ClusterManager) IsValid(cluster *api.Cluster) (bool, error) {
	return false, ErrNotImplemented
}

func (cm *ClusterManager) GetSSHConfig(cluster *api.Cluster, node *core.Node) (*api.SSHConfig, error) {
	cfg := &api.SSHConfig{
		PrivateKey: SSHKey(cm.ctx).PrivateKey,
		User:       "ubuntu",
		HostPort:   int32(22),
	}
	for _, addr := range node.Status.Addresses {
		if addr.Type == core.NodeExternalIP {
			cfg.HostIP = addr.Address
		}
	}
	if net.ParseIP(cfg.HostIP) == nil {
		return nil, errors.Errorf("failed to detect external Ip for node %s of cluster %s", node.Name, cluster.Name)
	}
	return cfg, nil
}

func (cm *ClusterManager) GetKubeConfig(cluster *api.Cluster) (*api.KubeConfig, error) {
	return nil, nil
}
