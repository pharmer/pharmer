package aws

import (
	"bytes"
	"context"
	"strings"

	api "github.com/appscode/pharmer/apis/v1alpha1"
	. "github.com/appscode/pharmer/cloud"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/cert"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1alpha1"
)

func newNodeTemplateData(ctx context.Context, cluster *api.Cluster, ng *api.NodeGroup, token string) TemplateData {
	td := TemplateData{
		ClusterName:      cluster.Name,
		KubeletVersion:   cluster.Spec.KubeletVersion,
		KubeadmVersion:   cluster.Spec.KubeletVersion,
		KubeadmToken:     token,
		CAKey:            string(cert.EncodePrivateKeyPEM(CAKey(ctx))),
		FrontProxyKey:    string(cert.EncodePrivateKeyPEM(FrontProxyCAKey(ctx))),
		APIServerAddress: cluster.APIServerAddress(),
		NetworkProvider:  cluster.Spec.Networking.NetworkProvider,
		Provider:         cluster.Spec.Cloud.CloudProvider,
		ExternalProvider: false, // AWS does not use out-of-tree CCM
		ExtraDomains:     strings.Join(cluster.Spec.APIServerCertSANs, ","),
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
		td.KubeletExtraArgs["cloud-provider"] = cluster.Spec.Cloud.CloudProvider // --cloud-config is not needed, since IAM is used. //with provider not working
	}
	return td
}

func newMasterTemplateData(ctx context.Context, cluster *api.Cluster, ng *api.NodeGroup) TemplateData {
	td := newNodeTemplateData(ctx, cluster, ng, "")
	td.KubeletExtraArgs["node-labels"] = api.NodeLabels{
		api.NodePoolKey: ng.Name,
	}.String()

	hostPath := kubeadmapi.HostPathMount{
		Name:      "cloud-config",
		HostPath:  "/etc/kubernetes/pharmer",
		MountPath: "/etc/kubernetes/pharmer",
	}
	cfg := kubeadmapi.MasterConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubeadm.k8s.io/v1alpha1",
			Kind:       "MasterConfiguration",
		},
		API: kubeadmapi.API{
			AdvertiseAddress: cluster.Spec.API.AdvertiseAddress,
			BindPort:         cluster.Spec.API.BindPort,
		},
		APIServerExtraVolumes:         []kubeadmapi.HostPathMount{hostPath},
		ControllerManagerExtraVolumes: []kubeadmapi.HostPathMount{hostPath},
		Networking: kubeadmapi.Networking{
			ServiceSubnet: cluster.Spec.Networking.ServiceSubnet,
			PodSubnet:     cluster.Spec.Networking.PodSubnet,
			DNSDomain:     cluster.Spec.Networking.DNSDomain,
		},
		KubernetesVersion:          cluster.Spec.KubernetesVersion,
		CloudProvider:              cluster.Spec.Cloud.CloudProvider,
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
{{ define "prepare-host" }}
NODE_NAME=$(curl http://169.254.169.254/2007-01-19/meta-data/local-hostname)
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
