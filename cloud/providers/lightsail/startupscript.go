package lightsail

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
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/pubkeypin"
)

func newNodeTemplateData(ctx context.Context, cluster *api.Cluster, ng *api.NodeGroup, nodeName, token string) TemplateData {
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
		ExternalProvider:  true, // DigitalOcean uses out-of-tree CCM

		NodeName: nodeName,
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
		td.KubeletExtraArgs["cloud-provider"] = "external" // --cloud-config is not needed

		if cluster.Spec.Cloud.CCMCredentialName == "" {
			panic(errors.New("no cloud controller manager credential found"))
		}
		cred, err := Store(ctx).Owner(owner).Credentials().Get(cluster.Spec.Cloud.CCMCredentialName)
		if err != nil {
			panic(err)
		}
		typed := credential.AWS{CommonSpec: credential.CommonSpec(cred.Spec)}
		if ok, err := typed.IsValid(); !ok {
			panic(err)
		}
		cloudConfig := &api.LightsailCloudConfig{
			AccessKeyID:     typed.AccessKeyID(),
			SecretAccessKey: typed.SecretAccessKey(),
		}
		data, err := json.Marshal(cloudConfig)
		if err != nil {
			panic(err)
		}
		td.CloudConfig = string(data)
	}
	return td
}

func newMasterTemplateData(ctx context.Context, cluster *api.Cluster, ng *api.NodeGroup, nodeName string) TemplateData {
	td := newNodeTemplateData(ctx, cluster, ng, nodeName, "")
	td.KubeletExtraArgs["node-labels"] = api.NodeLabels{
		api.NodePoolKey: ng.Name,
	}.String()

	ifg := kubeadmapi.InitConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubeadm.k8s.io/v1beta1",
			Kind:       "InitConfiguration",
		},
		NodeRegistration: kubeadmapi.NodeRegistrationOptions{
			KubeletExtraArgs: td.KubeletExtraArgs,
		},
		LocalAPIEndpoint: kubeadmapi.APIEndpoint{
			AdvertiseAddress: cluster.Spec.API.AdvertiseAddress,
			BindPort:         cluster.Spec.API.BindPort,
		},
	}
	td.InitConfiguration = &ifg
	cfg := kubeadmapi.ClusterConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubeadm.k8s.io/v1beta1",
			Kind:       "ClusterConfiguration",
		},
		Networking: kubeadmapi.Networking{
			ServiceSubnet: cluster.Spec.Networking.ServiceSubnet,
			PodSubnet:     cluster.Spec.Networking.PodSubnet,
			DNSDomain:     cluster.Spec.Networking.DNSDomain,
		},
		KubernetesVersion: cluster.Spec.KubernetesVersion,
		// "external": cloudprovider not supported for apiserver and controller-manager
		// https://github.com/kubernetes/kubernetes/pull/50545
		APIServer: kubeadmapi.APIServer{
			ControlPlaneComponent: kubeadmapi.ControlPlaneComponent{
				ExtraArgs: cluster.Spec.APIServerExtraArgs,
			},
			CertSANs: cluster.Spec.APIServerCertSANs,
		},
		ControllerManager: kubeadmapi.ControlPlaneComponent{
			ExtraArgs: cluster.Spec.ControllerManagerExtraArgs,
		},
		Scheduler: kubeadmapi.ControlPlaneComponent{
			ExtraArgs: cluster.Spec.SchedulerExtraArgs,
		},
		ClusterName: cluster.Name,
	}
	td.ClusterConfiguration = &cfg
	return td
}

var (
	customTemplate = `
{{ define "init-script" }}
export DEBIAN_FRONTEND=noninteractive
export DEBCONF_NONINTERACTIVE_SEEN=true

exec_until_success() {
	$1
	while [ $? -ne 0 ]; do
		sleep 2
		$1
	done
}
{{ end }}

{{ define "prepare-host" }}
hostnamectl set-hostname {{ .NodeName }}
{{ end }}
`
)

func (conn *cloudConnector) renderStartupScript(ng *api.NodeGroup, nodeName, token string) (string, error) {
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
		if err := tpl.ExecuteTemplate(&script, api.RoleMaster, newMasterTemplateData(conn.ctx, conn.cluster, ng, nodeName)); err != nil {
			return "", err
		}
	} else {
		if err := tpl.ExecuteTemplate(&script, api.RoleNode, newNodeTemplateData(conn.ctx, conn.cluster, ng, nodeName, token)); err != nil {
			return "", err
		}
	}
	return script.String(), nil
}
