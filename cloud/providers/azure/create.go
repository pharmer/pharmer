package azure

import (
	"encoding/base64"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/log"
	"github.com/pkg/errors"
	"gomodules.xyz/cert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"pharmer.dev/cloud/pkg/credential"
	api "pharmer.dev/pharmer/apis/v1alpha1"
	capiAzure "pharmer.dev/pharmer/apis/v1alpha1/azure"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) GetDefaultMachineProviderSpec(sku string, role api.MachineRole) (clusterapi.ProviderSpec, error) {
	cluster := cm.Cluster
	pubkey, privkey, err := cm.StoreProvider.SSHKeys(cluster.Name).Get(cluster.GenSSHKeyExternalID())
	if err != nil {
		return clusterapi.ProviderSpec{}, errors.Wrap(err, "failed to get ssh keys")
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
		SSHPublicKey:  base64.StdEncoding.EncodeToString(pubkey),
		SSHPrivateKey: base64.StdEncoding.EncodeToString(privkey),
	}

	rawSpec, err := capiAzure.EncodeMachineSpec(spec)
	if err != nil {
		return clusterapi.ProviderSpec{}, errors.Wrap(err, "failed to encode machine provider spec")
	}

	return clusterapi.ProviderSpec{
		Value: rawSpec,
	}, nil

}

// TODO: add test to make sure that apiserver certSANs have internal lb adress
func (cm *ClusterManager) SetDefaultCluster() error {
	cluster := cm.Cluster
	n := namer{cluster: cluster}
	config := &cluster.Spec.Config

	config.APIServerExtraArgs["cloud-config"] = "/etc/kubernetes/azure.json"
	config.APIServerCertSANs = append(config.APIServerCertSANs, DefaultInternalLBIPAddress)

	credentialName := cluster.Spec.Config.CredentialName
	cred, err := cm.StoreProvider.Credentials().Get(credentialName)
	if err != nil {
		return errors.Wrapf(err, "failed to get credential %q", credentialName)
	}
	typed := credential.Azure{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		log.Infof("Invalid credential: %v", err)
		return err
	}
	config.SSHUserName = "capi"

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

	return cm.SetClusterProviderConfig()
}

func (cm *ClusterManager) SetClusterProviderConfig() error {
	cluster := cm.Cluster
	n := namer{cluster: cluster}

	conf := &capiAzure.AzureClusterProviderSpec{
		TypeMeta: metav1.TypeMeta{
			APIVersion: api.AzureProviderGroupName + "/" + api.AzureProviderApiVersion,
			Kind:       api.AzureProviderClusterKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: cluster.Name,
		},
		NetworkSpec:   capiAzure.NetworkSpec{},
		ResourceGroup: n.ResourceGroupName(),
		Location:      cluster.Spec.Config.Cloud.Zone,

		SSHPublicKey:  base64.StdEncoding.EncodeToString(cm.Certs.SSHKey.PublicKey),
		SSHPrivateKey: base64.StdEncoding.EncodeToString(cm.Certs.SSHKey.PrivateKey),
		CAKeyPair: capiAzure.KeyPair{
			Cert: cert.EncodeCertPEM(cm.Certs.CACert.Cert),
			Key:  cert.EncodePrivateKeyPEM(cm.Certs.CACert.Key),
		},
		EtcdCAKeyPair: capiAzure.KeyPair{
			Cert: cert.EncodeCertPEM(cm.Certs.EtcdCACert.Cert),
			Key:  cert.EncodePrivateKeyPEM(cm.Certs.EtcdCACert.Key),
		},
		FrontProxyCAKeyPair: capiAzure.KeyPair{
			Cert: cert.EncodeCertPEM(cm.Certs.FrontProxyCACert.Cert),
			Key:  cert.EncodePrivateKeyPEM(cm.Certs.FrontProxyCACert.Key),
		},
		SAKeyPair: capiAzure.KeyPair{
			Cert: cert.EncodeCertPEM(cm.Certs.ServiceAccountCert.Cert),
			Key:  cert.EncodePrivateKeyPEM(cm.Certs.ServiceAccountCert.Key),
		},
	}

	rawSpec, err := capiAzure.EncodeClusterSpec(conf)
	if err != nil {
		return err
	}

	cluster.Spec.ClusterAPI.Spec.ProviderSpec.Value = rawSpec

	return nil
}
