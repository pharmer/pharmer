/*
Copyright The Pharmer Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package packet

import (
	"pharmer.dev/pharmer/cloud"

	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) NewNodeTemplateData(machine *clusterapi.Machine, token string, td cloud.TemplateData) cloud.TemplateData {
	td.ExternalProvider = true // Packet uses out-of-tree CCM

	// ref: https://kubernetes.io/docs/admin/kubeadm/#cloud-provider-integrations-experimental
	td.KubeletExtraArgs["cloud-provider"] = "external" // --cloud-config is not needed
	td.KubeletExtraArgs["enable-controller-attach-detach"] = "false"
	td.KubeletExtraArgs["keep-terminated-pod-volumes"] = "true"

	joinConf, _ := td.JoinConfigurationYAML()
	td.JoinConfiguration = joinConf
	return td
}

func (cm *ClusterManager) NewMasterTemplateData(machine *clusterapi.Machine, token string, td cloud.TemplateData) cloud.TemplateData {
	td.ClusterConfiguration = cloud.GetDefaultKubeadmClusterConfig(cm.Cluster, nil)

	return td
}

var (
	customTemplate = `
{{ define "init-os" }}
# Avoid using Packet's Ubuntu mirror
curl -fsSL --retry 5 -o /etc/apt/sources.list https://raw.githubusercontent.com/pharmer/addons/master/ubuntu/16.04/sources.list
{{ end }}
{{ define "prepare-host" }}
pre-k machine swapoff
{{ end }}

{{ define "install-storage-plugin" }}
# Deploy storage RBAC
cmd='kubectl apply --kubeconfig /etc/kubernetes/admin.conf -f https://raw.githubusercontent.com/pharmer/addons/release-1.13.1/cloud-storage/rbac.yaml'
exec_until_success "$cmd"

#Deploy plugin
cmd='kubectl apply --kubeconfig /etc/kubernetes/admin.conf -f https://raw.githubusercontent.com/pharmer/addons/release-1.13.1/cloud-storage/{{ .Provider }}/flexplugin.yaml'
exec_until_success "$cmd"

#Deploy provisioner
cmd='kubectl apply --kubeconfig /etc/kubernetes/admin.conf -f https://raw.githubusercontent.com/pharmer/addons/release-1.13.1/cloud-storage/{{ .Provider }}/provisioner.yaml'
exec_until_success "$cmd"
{{ end }}
`
)
