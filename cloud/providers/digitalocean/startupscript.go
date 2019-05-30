package digitalocean

import (
	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func newNodeTemplateData(cm *CloudManager, cluster *api.Cluster, machine *clusterapi.Machine, token string) TemplateData {
	td := NewNodeTemplateData(cm, cluster, machine, token)
	td.ExternalProvider = true // DigitalOcean uses out-of-tree CCM

	// ref: https://kubernetes.io/docs/admin/kubeadm/#cloud-provider-integrations-experimental
	td.KubeletExtraArgs["cloud-provider"] = "external" // --cloud-config is not needed
	//td.KubeletExtraArgs["enable-controller-attach-detach"] = "false"
	//td.KubeletExtraArgs["keep-terminated-pod-volumes"] = "true"

	joinConf, _ := td.JoinConfigurationYAML()
	td.JoinConfiguration = joinConf

	return td
}

func (cm *ClusterManager) NewMasterTemplateData(machine *clusterapi.Machine, token string, td TemplateData) TemplateData {
	hostPath := kubeadmapi.HostPathMount{}

	td.ClusterConfiguration = GetDefaultKubeadmClusterConfig(cm.Cluster, hostPath)

	return td
}

func (conn *cloudConnector) renderStartupScript(cluster *api.Cluster, machine *clusterapi.Machine, token string) (string, error) {
	return RenderStartupScript(machine, customTemplate, newMasterTemplateData(conn.CloudManager, cluster, machine, token), newNodeTemplateData(conn.CloudManager, cluster, machine, token))
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
  until getent hosts $(hostname -i || echo _error_) &>/dev/null; do
    echo 'Waiting for functional DNS (trying to resolve my own IP)...'
    sleep 3
  done

  echo "Networking functional on $(hostname) ($(hostname -i))"
}

ensure_basic_networking
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

#Deploy initializer
cmd='kubectl apply --kubeconfig /etc/kubernetes/admin.conf -f https://raw.githubusercontent.com/pharmer/addons/release-1.13.1/cloud-controller-manager/initializer.yaml'
exec_until_success "$cmd"
{{ end }}


{{ define "prepare-host" }}
NODE_NAME=$(hostname)
{{ end }}
`
)
