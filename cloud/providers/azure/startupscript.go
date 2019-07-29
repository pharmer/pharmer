package azure

import (
	"encoding/json"

	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
	"pharmer.dev/cloud/pkg/credential"
	api "pharmer.dev/pharmer/apis/v1alpha1"
	"pharmer.dev/pharmer/cloud"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) NewNodeTemplateData(machine *v1alpha1.Machine, token string, td cloud.TemplateData) cloud.TemplateData {
	cluster := cm.Cluster
	td.ExternalProvider = false // Azure does not use out-of-tree CCM
	// ref: https://kubernetes.io/docs/admin/kubeadm/#cloud-provider-integrations-experimental
	td.KubeletExtraArgs["cloud-provider"] = "azure" // requires --cloud-config

	cred, err := cm.GetCredential()
	if err != nil {
		panic(err)
	}
	typed := credential.Azure{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		panic(err)
	}

	namer := namer{cluster: cluster}
	cloudConfig := &api.AzureCloudConfig{
		Cloud:                        "AzurePublicCloud",
		TenantID:                     typed.TenantID(),
		SubscriptionID:               typed.SubscriptionID(),
		AadClientID:                  typed.ClientID(),
		AadClientSecret:              typed.ClientSecret(),
		ResourceGroup:                cluster.ClusterConfig().Cloud.Azure.ResourceGroup,
		Location:                     cluster.ClusterConfig().Cloud.Zone,
		VMType:                       "standard",
		SubnetName:                   namer.GenerateNodeSubnetName(),
		SecurityGroupName:            namer.GenerateNodeSecurityGroupName(),
		VnetName:                     namer.VirtualNetworkName(),
		RouteTableName:               namer.RouteTableName(),
		PrimaryAvailabilitySetName:   "",
		PrimaryScaleSetName:          "",
		CloudProviderBackoff:         true,
		CloudProviderBackoffRetries:  6,
		CloudProviderBackoffExponent: 1.5,
		CloudProviderBackoffDuration: 5,
		CloudProviderBackoffJitter:   1.0,
		CloudProviderRatelimit:       true,
		CloudProviderRateLimitQPS:    3.0,
		CloudProviderRateLimitBucket: 10,
		UseManagedIdentityExtension:  false,
		UserAssignedIdentityID:       "",
		UseInstanceMetadata:          true,
		LoadBalancerSku:              "Standard",
		ExcludeMasterFromStandardLB:  true,
		ProviderVaultName:            "",
		MaximumLoadBalancerRuleCount: 250,
		ProviderKeyName:              "k8s",
		ProviderKeyVersion:           "",
	}
	data, err := json.MarshalIndent(cloudConfig, "", "  ")
	if err != nil {
		panic(err)
	}
	td.CloudConfig = string(data)

	// ref: https://github.com/kubernetes/kubernetes/blob/1910086bbce4f08c2b3ab0a4c0a65c913d4ec921/cmd/kubeadm/app/phases/controlplane/manifests.go#L41
	td.KubeletExtraArgs["cloud-config"] = "/etc/kubernetes/azure.json"

	return td
}

func (cm *ClusterManager) NewMasterTemplateData(machine *clusterapi.Machine, token string, td cloud.TemplateData) cloud.TemplateData {
	hostPath := kubeadmapi.HostPathMount{
		Name:      "cloud-config",
		HostPath:  "/etc/kubernetes/azure.json",
		MountPath: "/etc/kubernetes/azure.json",
	}

	td.ClusterConfiguration = cloud.GetDefaultKubeadmClusterConfig(cm.Cluster, &hostPath)

	return td
}

var (
	customTemplate = `
{{ define "init-os" }}
# We rely on DNS for a lot, and it's just not worth doing a whole lot of startup work if this isn't ready yet.
# ref: https://github.com/kubernetes/kubernetes/blob/443908193d564736d02efdca4c9ba25caf1e96fb/cluster/gce/configure-vm.sh#L24
ensure_basic_networking() {
  until getent hosts $(hostname -f || echo _error_) &>/dev/null; do
    echo 'Waiting for functional DNS (trying to resolve my own FQDN)...'
    sleep 3
  done

  echo "Networking functional on $(hostname) ($(hostname -i))"
}

ensure_basic_networking

{{ end }}

{{ define "cloud-config" }}
cat > /etc/kubernetes/azure.json <<EOF
{{ .CloudConfig }}
EOF
{{ end }}
`
)
