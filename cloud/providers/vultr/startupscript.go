package vultr

import (
	"bytes"
	"context"
	"strings"

	api "github.com/appscode/pharmer/apis/v1alpha1"
	. "github.com/appscode/pharmer/cloud"
	"github.com/hashicorp/go-version"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/cert"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1alpha1"
)

func newNodeTemplateData(ctx context.Context, cluster *api.Cluster, ng *api.NodeGroup, token string) TemplateData {
	td := TemplateData{
		KubernetesVersion: cluster.Spec.KubernetesVersion,
		KubeadmVersion:    cluster.Spec.MasterKubeadmVersion,
		KubeadmToken:      token,
		CAKey:             string(cert.EncodePrivateKeyPEM(CAKey(ctx))),
		FrontProxyKey:     string(cert.EncodePrivateKeyPEM(FrontProxyCAKey(ctx))),
		APIServerAddress:  cluster.APIServerAddress(),
		APIBindPort:       6443,
		ExtraDomains:      cluster.Spec.ClusterExternalDomain,
		NetworkProvider:   cluster.Spec.Networking.NetworkProvider,
		Provider:          cluster.Spec.Cloud.CloudProvider,
		ExternalProvider:  true, // Vultr uses out-of-tree CCM
	}
	{
		extraDomains := []string{}
		if domain := Extra(ctx).ExternalDomain(cluster.Name); domain != "" {
			extraDomains = append(extraDomains, domain)
		}
		if domain := Extra(ctx).InternalDomain(cluster.Name); domain != "" {
			extraDomains = append(extraDomains, domain)
		}
		td.ExtraDomains = strings.Join(extraDomains, ",")
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
			api.NodeLabelKey_NodeGroup: ng.Name,
			api.RoleNodeKey:            "",
		}.String()
		// ref: https://kubernetes.io/docs/admin/kubeadm/#cloud-provider-integrations-experimental
		td.KubeletExtraArgs["cloud-provider"] = "external" // --cloud-config is not needed
	}
	return td
}

func newMasterTemplateData(ctx context.Context, cluster *api.Cluster, ng *api.NodeGroup) TemplateData {
	td := newNodeTemplateData(ctx, cluster, ng, "")
	td.KubeletExtraArgs["node-labels"] = api.NodeLabels{
		api.NodeLabelKey_NodeGroup: ng.Name,
	}.String()

	if cluster.Spec.MasterKubeadmVersion != "" {
		if v, err := version.NewVersion(cluster.Spec.MasterKubeadmVersion); err == nil && v.Prerelease() != "" {
			td.IsPreReleaseVersion = true
		} else {
			if lv, err := GetLatestKubeadmVerson(); err == nil && lv == cluster.Spec.MasterKubeadmVersion {
				td.KubeadmVersion = ""
			}
		}
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
		Networking: kubeadmapi.Networking{
			ServiceSubnet: cluster.Spec.Networking.ServiceSubnet,
			PodSubnet:     cluster.Spec.Networking.PodSubnet,
			DNSDomain:     cluster.Spec.Networking.DNSDomain,
		},
		KubernetesVersion:          cluster.Spec.KubernetesVersion,
		CloudProvider:              "external",
		APIServerExtraArgs:         map[string]string{},
		ControllerManagerExtraArgs: map[string]string{},
		SchedulerExtraArgs:         map[string]string{},
		APIServerCertSANs:          []string{},
	}
	td.MasterConfiguration = &cfg
	return td
}

var (
	customTemplate = `
{{ define "prepare-host" }}
# https://www.vultr.com/docs/configuring-private-network
PRIVATE_ADDRESS=$(/usr/bin/curl http://169.254.169.254/v1/interfaces/1/ipv4/address 2> /dev/null)
PRIVATE_NETMASK=$(/usr/bin/curl http://169.254.169.254/v1/interfaces/1/ipv4/netmask 2> /dev/null)
/bin/cat >>/etc/network/interfaces <<EOF

auto eth1
iface eth1 inet static
    address $PRIVATE_ADDRESS
    netmask $PRIVATE_NETMASK
            mtu 1450
EOF
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
