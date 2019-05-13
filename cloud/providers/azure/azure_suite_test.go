package azure

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	cloudapi "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	"github.com/pharmer/cloud/pkg/credential"
	"github.com/pharmer/pharmer/apis/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func TestAzure(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Azure Suite")
}

func getCluster() *v1beta1.Cluster {
	return &v1beta1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: v1beta1.PharmerClusterSpec{
			ClusterAPI: &clusterapi.Cluster{},
			Config: &v1beta1.ClusterConfig{
				Cloud: v1beta1.CloudSpec{
					NetworkProvider: v1beta1.PodNetworkCalico,
					CloudProvider:   "azure",
					Zone:            "eastus2",
				},
				CredentialName:    "test",
				KubernetesVersion: "v1.14.0",
			},
		},
	}
}

func getCredential() *cloudapi.Credential {
	return &cloudapi.Credential{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: cloudapi.CredentialSpec{
			Provider: "azure",
			Data: map[string]string{
				credential.AzureTenantID:       "tenantid",
				credential.AzureSubscriptionID: "subid",
				credential.AzureClientID:       "clientid",
				credential.AzureClientSecret:   "clientsecret",
			},
		},
	}
}

func getMachine() *clusterapi.Machine {
	return &clusterapi.Machine{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: clusterapi.MachineSpec{
			ObjectMeta:   metav1.ObjectMeta{},
			Taints:       nil,
			ProviderSpec: clusterapi.ProviderSpec{},
			Versions: clusterapi.MachineVersionInfo{
				Kubelet:      "v1.14.0",
				ControlPlane: "v1.14.0",
			},
			ConfigSource: nil,
			ProviderID:   nil,
		},
		Status: clusterapi.MachineStatus{},
	}
}
