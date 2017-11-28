package lightsail

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/appscode/errors"
	api "github.com/appscode/pharmer/apis/v1alpha1"
	. "github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/credential"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/cert"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1alpha1"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/pubkeypin"
)

func newNodeTemplateData(ctx context.Context, cluster *api.Cluster, ng *api.NodeGroup, token string) TemplateData {
	td := TemplateData{
		ClusterName:      cluster.Name,
		KubeletVersion:   cluster.Spec.KubeletVersion,
		KubeadmVersion:   cluster.Spec.KubeletVersion,
		KubeadmToken:     token,
		CAHash:           pubkeypin.Hash(CACert(ctx)),
		CAKey:            string(cert.EncodePrivateKeyPEM(CAKey(ctx))),
		FrontProxyKey:    string(cert.EncodePrivateKeyPEM(FrontProxyCAKey(ctx))),
		APIServerAddress: cluster.APIServerAddress(),
		NetworkProvider:  cluster.Spec.Networking.NetworkProvider,
		Provider:         cluster.Spec.Cloud.CloudProvider,
		ExternalProvider: true, // DigitalOcean uses out-of-tree CCM
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
		cred, err := Store(ctx).Credentials().Get(cluster.Spec.Cloud.CCMCredentialName)
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

func newMasterTemplateData(ctx context.Context, cluster *api.Cluster, ng *api.NodeGroup) TemplateData {
	td := newNodeTemplateData(ctx, cluster, ng, "")
	td.KubeletExtraArgs["node-labels"] = api.NodeLabels{
		api.NodePoolKey: ng.Name,
	}.String()

	cfg := kubeadmapi.MasterConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubeadm.k8s.io/v1alpha1",
			Kind:       "MasterConfiguration",
		},
		API: kubeadmapi.API{
			AdvertiseAddress: cluster.Spec.API.AdvertiseAddress,
			BindPort:         cluster.Spec.API.BindPort,
		},
		Networking: kubeadmapi.Networking{
			ServiceSubnet: cluster.Spec.Networking.ServiceSubnet,
			PodSubnet:     cluster.Spec.Networking.PodSubnet,
			DNSDomain:     cluster.Spec.Networking.DNSDomain,
		},
		KubernetesVersion: cluster.Spec.KubernetesVersion,
		// "external": cloudprovider not supported for apiserver and controller-manager
		// https://github.com/kubernetes/kubernetes/pull/50545
		CloudProvider:              "",
		APIServerExtraArgs:         cluster.Spec.APIServerExtraArgs,
		ControllerManagerExtraArgs: cluster.Spec.ControllerManagerExtraArgs,
		SchedulerExtraArgs:         cluster.Spec.SchedulerExtraArgs,
		APIServerCertSANs:          cluster.Spec.APIServerCertSANs,
	}
	td.MasterConfiguration = &cfg
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
`
)

func (conn *cloudConnector) renderStartupScript(ng *api.NodeGroup, token string) (string, error) {
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
		if err := tpl.ExecuteTemplate(&script, api.RoleMaster, newMasterTemplateData(conn.ctx, conn.cluster, ng)); err != nil {
			return "", err
		}
	} else {
		if err := tpl.ExecuteTemplate(&script, api.RoleNode, newNodeTemplateData(conn.ctx, conn.cluster, ng, token)); err != nil {
			return "", err
		}
	}
	return script.String(), nil
}
