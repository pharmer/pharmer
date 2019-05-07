package azure

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/pharmer/cloud/pkg/credential"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/cert"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/pubkeypin"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/util"
)

func newNodeTemplateData(ctx context.Context, cluster *api.Cluster, machine *clusterapi.Machine, owner, token string) TemplateData {
	td := TemplateData{
		ClusterName:       cluster.Name,
		KubernetesVersion: machine.Spec.Versions.ControlPlane,
		KubeadmToken:      token,
		CAHash:            pubkeypin.Hash(CACert(ctx)),
		CAKey:             string(cert.EncodePrivateKeyPEM(CAKey(ctx))),
		FrontProxyKey:     string(cert.EncodePrivateKeyPEM(FrontProxyCAKey(ctx))),
		SAKey:             string(cert.EncodePrivateKeyPEM(SaKey(ctx))),
		ETCDCAKey:         string(cert.EncodePrivateKeyPEM(EtcdCaKey(ctx))),
		APIServerAddress:  cluster.APIServerAddress(),
		NetworkProvider:   cluster.ClusterConfig().Cloud.NetworkProvider,
		Provider:          cluster.ClusterConfig().Cloud.CloudProvider,
		ExternalProvider:  false, // Azure does not use out-of-tree CCM
	}
	{
		td.KubeletExtraArgs = map[string]string{}
		for k, v := range cluster.Spec.Config.KubeletExtraArgs {
			td.KubeletExtraArgs[k] = v
		}
		/*for k, v := range machine.Spec.Template.Spec.KubeletExtraArgs {
			td.KubeletExtraArgs[k] = v
		}*/
		td.KubeletExtraArgs["node-labels"] = api.NodeLabels{
			api.NodePoolKey: machine.Name,
			api.RoleNodeKey: "",
		}.String()
		// ref: https://kubernetes.io/docs/admin/kubeadm/#cloud-provider-integrations-experimental
		td.KubeletExtraArgs["cloud-provider"] = "azure" // requires --cloud-config
		if cluster.Spec.Config.Cloud.CCMCredentialName == "" {
			panic(errors.New("no cloud controller manager credential found"))
		}

		cred, err := Store(ctx).Owner(owner).Credentials().Get(cluster.Spec.Config.Cloud.CCMCredentialName)
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

		// Kubeadm will send cloud-config to kube-apiserver and kube-controller-manager
		// ref: https://github.com/kubernetes/kubernetes/blob/1910086bbce4f08c2b3ab0a4c0a65c913d4ec921/cmd/kubeadm/app/phases/controlplane/manifests.go#L193
		// ref: https://github.com/kubernetes/kubernetes/blob/1910086bbce4f08c2b3ab0a4c0a65c913d4ec921/cmd/kubeadm/app/phases/controlplane/manifests.go#L230

	}
	return td
}

func newMasterTemplateData(ctx context.Context, cluster *api.Cluster, machine *clusterapi.Machine, owner string) TemplateData {
	td := newNodeTemplateData(ctx, cluster, machine, owner, "")
	td.KubeletExtraArgs["node-labels"] = api.NodeLabels{
		api.NodePoolKey: machine.Name,
	}.String()

	hostPath := kubeadmapi.HostPathMount{
		Name:      "cloud-config",
		HostPath:  "/etc/kubernetes/azure.json",
		MountPath: "/etc/kubernetes/azure.json",
	}
	ifg := kubeadmapi.InitConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubeadm.k8s.io/v1beta1",
			Kind:       "InitConfiguration",
		},

		NodeRegistration: kubeadmapi.NodeRegistrationOptions{
			KubeletExtraArgs: td.KubeletExtraArgs,
		},
		LocalAPIEndpoint: kubeadmapi.APIEndpoint{
			//AdvertiseAddress: cluster.Spec.API.AdvertiseAddress,
			BindPort: 6443, //         cluster.Spec.API.BindPort,
		},
	}
	td.InitConfiguration = &ifg

	cfg := kubeadmapi.ClusterConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubeadm.k8s.io/v1beta1",
			Kind:       "ClusterConfiguration",
		},
		APIServer: kubeadmapi.APIServer{
			ControlPlaneComponent: kubeadmapi.ControlPlaneComponent{
				ExtraVolumes: []kubeadmapi.HostPathMount{hostPath},
				ExtraArgs:    cluster.Spec.Config.APIServerExtraArgs,
			},
			CertSANs: cluster.Spec.Config.APIServerCertSANs,
		},
		ControllerManager: kubeadmapi.ControlPlaneComponent{
			ExtraVolumes: []kubeadmapi.HostPathMount{hostPath},
			ExtraArgs:    cluster.Spec.Config.ControllerManagerExtraArgs,
		},
		Scheduler: kubeadmapi.ControlPlaneComponent{
			ExtraArgs: cluster.Spec.Config.SchedulerExtraArgs,
		},

		Networking: kubeadmapi.Networking{
			ServiceSubnet: cluster.Spec.ClusterAPI.Spec.ClusterNetwork.Services.CIDRBlocks[0],
			PodSubnet:     cluster.Spec.ClusterAPI.Spec.ClusterNetwork.Pods.CIDRBlocks[0],
			DNSDomain:     cluster.Spec.ClusterAPI.Spec.ClusterNetwork.ServiceDomain,
		},
		KubernetesVersion: cluster.Spec.Config.KubernetesVersion,
	}
	td.ControlPlaneEndpointsFromLB(&cfg, cluster)

	cfg.APIServer.CertSANs = append(cfg.APIServer.CertSANs, cluster.Spec.Config.Cloud.Azure.InternalLBIPAddress)

	td.ClusterConfiguration = &cfg
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
`
)

func (conn *cloudConnector) renderStartupScript(cluster *api.Cluster, machine *clusterapi.Machine, owner, token string) (string, error) {
	tpl, err := StartupScriptTemplate.Clone()
	if err != nil {
		return "", err
	}
	tpl, err = tpl.Parse(customTemplate)
	if err != nil {
		return "", err
	}

	var script bytes.Buffer
	if util.IsControlPlaneMachine(machine) {
		if err := tpl.ExecuteTemplate(&script, api.RoleMaster, newMasterTemplateData(conn.ctx, conn.cluster, machine, owner)); err != nil {
			return "", err
		}
	} else {
		if err := tpl.ExecuteTemplate(&script, api.RoleNode, newNodeTemplateData(conn.ctx, conn.cluster, machine, owner, token)); err != nil {
			return "", err
		}
	}
	return script.String(), nil
}
