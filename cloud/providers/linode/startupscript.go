package linode

import (
	"pharmer.dev/pharmer/cloud"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) NewNodeTemplateData(machine *clusterapi.Machine, token string, td cloud.TemplateData) cloud.TemplateData {
	td.ExternalProvider = true // Linode uses out-of-tree CCM

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

#<UDF name="hostname" label="The hostname for the new Linode.">
# HOSTNAME=

echo $HOSTNAME > /etc/hostname
hostname -F /etc/hostname

IPADDR=$(/sbin/ifconfig eth0 | awk '/inet / { print $2 }' | sed 's/addr://')
echo $IPADDR $HOSTNAME >> /etc/hosts

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

# Avoid using Linode's Ubuntu mirror
curl -fsSL --retry 5 -o /etc/apt/sources.list https://raw.githubusercontent.com/pharmer/addons/master/ubuntu/16.04/sources.list

# http://ask.xmodulo.com/disable-ipv6-linux.html
/bin/cat >>/etc/sysctl.conf <<EOF
# to disable IPv6 on all interfaces system wide
net.ipv6.conf.all.disable_ipv6 = 1

# to disable IPv6 on a specific interface (e.g., eth0, lo)
net.ipv6.conf.lo.disable_ipv6 = 1
net.ipv6.conf.eth0.disable_ipv6 = 1
EOF
/sbin/sysctl -p /etc/sysctl.conf
/bin/sed -i 's/^#AddressFamily any/AddressFamily inet/' /etc/ssh/sshd_config
{{ end }}

{{ define "prepare-host" }}
#HOSTNAME=$(pre-k linode hostname -k {{ .ClusterName }})
#hostnamectl set-hostname $HOSTNAME
# ref: https://bugs.debian.org/cgi-bin/bugreport.cgi?bug=769356
# ref: https://github.com/kubernetes/kubernetes/blob/82c986ecbcdf99a87cd12a7e2cf64f90057b9acd/cmd/kubeadm/app/preflight/checks.go#L927
touch /lib/modules/$(uname -r)/modules.builtin
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
`
)
