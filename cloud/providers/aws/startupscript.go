package aws

import (
	"bytes"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/util"
)

func newNodeTemplateData(conn *cloudConnector, cluster *api.Cluster, machine clusterapi.Machine, token string) TemplateData {
	td := NewNodeTemplateData(conn, cluster, machine, token)
	td.ExternalProvider = false // AWS does not use out-of-tree CCM

	// ref: https://kubernetes.io/docs/admin/kubeadm/#cloud-provider-integrations-experimental
	td.KubeletExtraArgs["cloud-provider"] = cluster.Spec.Config.Cloud.CloudProvider // --cloud-config is not needed, since IAM is used. //with provider not working

	return td
}

func newMasterTemplateData(conn *cloudConnector, cluster *api.Cluster, machine *clusterapi.Machine) TemplateData {
	td := newNodeTemplateData(conn, cluster, *machine, "")
	td.KubeletExtraArgs["node-labels"] = api.NodeLabels{
		api.NodePoolKey: machine.Name,
	}.String()

	hostPath := kubeadmapi.HostPathMount{
		Name:      "cloud-config",
		HostPath:  "/etc/kubernetes/ccm",
		MountPath: "/etc/kubernetes/ccm",
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
			BindPort: 6443,
			// AdvertiseAddress: cluster.Spec.Config.API.AdvertiseAddress,
			// BindPort:         cluster.Spec.Config.API.BindPort,
		},
	}
	td.InitConfiguration = &ifg

	cfg := kubeadmapi.ClusterConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubeadm.k8s.io/v1beta1",
			Kind:       "ClusterConfiguration",
		},

		Networking: kubeadmapi.Networking{
			ServiceSubnet: cluster.Spec.ClusterAPI.Spec.ClusterNetwork.Services.CIDRBlocks[0],
			PodSubnet:     cluster.Spec.ClusterAPI.Spec.ClusterNetwork.Pods.CIDRBlocks[0],
			DNSDomain:     cluster.Spec.ClusterAPI.Spec.ClusterNetwork.ServiceDomain,
		},
		KubernetesVersion: cluster.Spec.Config.KubernetesVersion,
		//CloudProvider:              cluster.Spec.Cloud.CloudProvider,

		APIServer: kubeadmapi.APIServer{
			ControlPlaneComponent: kubeadmapi.ControlPlaneComponent{
				ExtraArgs:    cluster.Spec.Config.APIServerExtraArgs,
				ExtraVolumes: []kubeadmapi.HostPathMount{hostPath},
			},
			CertSANs: cluster.Spec.Config.APIServerCertSANs,
		},
		ControllerManager: kubeadmapi.ControlPlaneComponent{
			ExtraArgs: map[string]string{
				"cloud-provider": cluster.Spec.Config.Cloud.CloudProvider,
			},
			ExtraVolumes: []kubeadmapi.HostPathMount{hostPath},
		},
		Scheduler: kubeadmapi.ControlPlaneComponent{
			ExtraArgs: cluster.Spec.Config.SchedulerExtraArgs,
		},
	}
	td.ControlPlaneEndpointsFromLB(&cfg, cluster)

	td.ClusterConfiguration = &cfg
	return td
}

var (
	customTemplate = `
{{ define "prepare-host" }}
NODE_NAME=$(curl http://169.254.169.254/2007-01-19/meta-data/local-hostname)
{{ end }}

`
)

// TODO(tahsin): works with pharmer-cli, doesn't work with pharm, why?
/*{{ define "mount-master-pd" }}
pre-k mount-master-pd --provider=aws
{{ end }}*/

func (conn *cloudConnector) renderStartupScript(machine *clusterapi.Machine, token string) (string, error) {
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
		if err := tpl.ExecuteTemplate(&script, api.RoleMaster, newMasterTemplateData(conn, conn.cluster, machine)); err != nil {
			return "", err
		}
	} else {
		if err := tpl.ExecuteTemplate(&script, api.RoleNode, newNodeTemplateData(conn, conn.cluster, *machine, token)); err != nil {
			return "", err
		}
	}
	return script.String(), nil
}
