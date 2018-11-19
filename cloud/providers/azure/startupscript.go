package azure

import (
	"bytes"
	"context"
	"encoding/json"

	api "github.com/pharmer/pharmer/apis/v1alpha1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/credential"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/cert"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1alpha3"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/pubkeypin"
)

func newNodeTemplateData(ctx context.Context, cluster *api.Cluster, ng *api.NodeGroup, owner, token string) TemplateData {
	td := TemplateData{
		ClusterName:       cluster.Name,
		KubernetesVersion: cluster.Spec.KubernetesVersion,
		KubeadmToken:      token,
		CAHash:            pubkeypin.Hash(CACert(ctx)),
		CAKey:             string(cert.EncodePrivateKeyPEM(CAKey(ctx))),
		FrontProxyKey:     string(cert.EncodePrivateKeyPEM(FrontProxyCAKey(ctx))),
		APIServerAddress:  cluster.APIServerAddress(),
		NetworkProvider:   cluster.Spec.Networking.NetworkProvider,
		Provider:          cluster.Spec.Cloud.CloudProvider,
		ExternalProvider:  false, // Azure does not use out-of-tree CCM
	}
	{
		td.KubeletExtraArgs = map[string]string{}
		for k, v := range cluster.Spec.KubeletExtraArgs {
			td.KubeletExtraArgs[k] = v
		}
		for k, v := range ng.Spec.Template.Spec.KubeletExtraArgs {
			td.KubeletExtraArgs[k] = v
		}
		td.KubeletExtraArgs["node-labels"] = api.NodeLabels{
			api.NodePoolKey: ng.Name,
			api.RoleNodeKey: "",
		}.String()
		// ref: https://kubernetes.io/docs/admin/kubeadm/#cloud-provider-integrations-experimental
		td.KubeletExtraArgs["cloud-provider"] = "azure" // requires --cloud-config
		if cluster.Spec.Cloud.CCMCredentialName == "" {
			panic(errors.New("no cloud controller manager credential found"))
		}

		cred, err := Store(ctx).Owner(owner).Credentials().Get(cluster.Spec.Cloud.CCMCredentialName)
		if err != nil {
			panic(err)
		}
		typed := credential.Azure{CommonSpec: credential.CommonSpec(cred.Spec)}
		if ok, err := typed.IsValid(); !ok {
			panic(err)
		}
		cloudConfig := &api.AzureCloudConfig{
			TenantID:           typed.TenantID(),
			SubscriptionID:     typed.SubscriptionID(),
			AadClientID:        typed.ClientID(),
			AadClientSecret:    typed.ClientSecret(),
			ResourceGroup:      cluster.Spec.Cloud.Azure.ResourceGroup,
			Location:           cluster.Spec.Cloud.Zone,
			SubnetName:         cluster.Spec.Cloud.Azure.SubnetName,
			SecurityGroupName:  cluster.Spec.Cloud.Azure.SecurityGroupName,
			VnetName:           cluster.Spec.Cloud.Azure.VnetName,
			RouteTableName:     cluster.Spec.Cloud.Azure.RouteTableName,
			StorageAccountName: cluster.Spec.Cloud.Azure.StorageAccountName,
		}
		data, err := json.MarshalIndent(cloudConfig, "", "  ")
		if err != nil {
			panic(err)
		}
		td.CloudConfig = string(data)

		// ref: https://github.com/kubernetes/kubernetes/blob/1910086bbce4f08c2b3ab0a4c0a65c913d4ec921/cmd/kubeadm/app/phases/controlplane/manifests.go#L41
		td.KubeletExtraArgs["cloud-config"] = "/etc/kubernetes/ccm/cloud-config"

		// Kubeadm will send cloud-config to kube-apiserver and kube-controller-manager
		// ref: https://github.com/kubernetes/kubernetes/blob/1910086bbce4f08c2b3ab0a4c0a65c913d4ec921/cmd/kubeadm/app/phases/controlplane/manifests.go#L193
		// ref: https://github.com/kubernetes/kubernetes/blob/1910086bbce4f08c2b3ab0a4c0a65c913d4ec921/cmd/kubeadm/app/phases/controlplane/manifests.go#L230

	}
	return td
}

func newMasterTemplateData(ctx context.Context, cluster *api.Cluster, ng *api.NodeGroup, owner string) TemplateData {
	td := newNodeTemplateData(ctx, cluster, ng, owner, "")
	td.KubeletExtraArgs["node-labels"] = api.NodeLabels{
		api.NodePoolKey: ng.Name,
	}.String()

	hostPath := kubeadmapi.HostPathMount{
		Name:      "cloud-config",
		HostPath:  "/etc/kubernetes/ccm",
		MountPath: "/etc/kubernetes/ccm",
	}
	ifg := kubeadmapi.InitConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubeadm.k8s.io/v1alpha3",
			Kind:       "InitConfiguration",
		},

		NodeRegistration: kubeadmapi.NodeRegistrationOptions{
			KubeletExtraArgs: td.KubeletExtraArgs,
		},
		APIEndpoint: kubeadmapi.APIEndpoint{
			AdvertiseAddress: cluster.Spec.API.AdvertiseAddress,
			BindPort:         cluster.Spec.API.BindPort,
		},
	}
	td.InitConfiguration = &ifg

	cfg := kubeadmapi.ClusterConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubeadm.k8s.io/v1alpha3",
			Kind:       "ClusterConfiguration",
		},

		APIServerExtraVolumes:         []kubeadmapi.HostPathMount{hostPath},
		ControllerManagerExtraVolumes: []kubeadmapi.HostPathMount{hostPath},
		Networking: kubeadmapi.Networking{
			ServiceSubnet: cluster.Spec.Networking.ServiceSubnet,
			PodSubnet:     cluster.Spec.Networking.PodSubnet,
			DNSDomain:     cluster.Spec.Networking.DNSDomain,
		},
		KubernetesVersion: cluster.Spec.KubernetesVersion,
		//CloudProvider:              cluster.Spec.Cloud.CloudProvider,
		APIServerExtraArgs:         cluster.Spec.APIServerExtraArgs,
		ControllerManagerExtraArgs: cluster.Spec.ControllerManagerExtraArgs,
		SchedulerExtraArgs:         cluster.Spec.SchedulerExtraArgs,
		APIServerCertSANs:          cluster.Spec.APIServerCertSANs,
	}

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

func (conn *cloudConnector) renderStartupScript(ng *api.NodeGroup, owner, token string) (string, error) {
	tpl, err := StartupScriptTemplate.Clone()
	if err != nil {
		return "", err
	}
	tpl, err = tpl.Parse(customTemplate)
	if err != nil {
		return "", err
	}

	var script bytes.Buffer
	if ng.Role() == api.RoleMaster {
		if err := tpl.ExecuteTemplate(&script, api.RoleMaster, newMasterTemplateData(conn.ctx, conn.cluster, ng, owner)); err != nil {
			return "", err
		}
	} else {
		if err := tpl.ExecuteTemplate(&script, api.RoleNode, newNodeTemplateData(conn.ctx, conn.cluster, ng, owner, token)); err != nil {
			return "", err
		}
	}
	return script.String(), nil
}
