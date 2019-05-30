package aws

import (
	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) NewNodeTemplateData(cluster *api.Cluster, machine *clusterapi.Machine, token string) TemplateData {
	td := NewNodeTemplateData(cm, machine, token)
	td.ExternalProvider = false // AWS does not use out-of-tree CCM

	// ref: https://kubernetes.io/docs/admin/kubeadm/#cloud-provider-integrations-experimental
	td.KubeletExtraArgs["cloud-provider"] = cluster.Spec.Config.Cloud.CloudProvider // --cloud-config is not needed, since IAM is used. //with provider not working

	return td
}

func (cm *ClusterManager) NewMasterTemplateData(machine *clusterapi.Machine, token string, td TemplateData) TemplateData {
	hostPath := kubeadmapi.HostPathMount{
		Name:      "cloud-config",
		HostPath:  "/etc/kubernetes/ccm",
		MountPath: "/etc/kubernetes/ccm",
	}

	td.ClusterConfiguration = GetDefaultKubeadmClusterConfig(cm.Cluster, hostPath)

	td.ClusterConfiguration.ControllerManager.ExtraArgs = map[string]string{
		"cloud-provider": cm.Cluster.Spec.Config.Cloud.CloudProvider,
	}

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
