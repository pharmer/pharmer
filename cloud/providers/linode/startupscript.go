package linode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"github.com/pharmer/cloud/pkg/credential"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/cert"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/pubkeypin"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/util"
)

func newNodeTemplateData(ctx context.Context, cluster *api.Cluster, machine *clusterv1.Machine, token string, owner string) TemplateData {
	td := TemplateData{
		ClusterName:       cluster.Name,
		KubernetesVersion: cluster.ClusterConfig().KubernetesVersion,
		KubeadmToken:      token,
		CAHash:            pubkeypin.Hash(CACert(ctx)),
		CAKey:             string(cert.EncodePrivateKeyPEM(CAKey(ctx))),
		SAKey:             string(cert.EncodePrivateKeyPEM(SaKey(ctx))),
		FrontProxyKey:     string(cert.EncodePrivateKeyPEM(FrontProxyCAKey(ctx))),
		ETCDCAKey:         string(cert.EncodePrivateKeyPEM(EtcdCaKey(ctx))),
		APIServerAddress:  cluster.APIServerAddress(),
		NetworkProvider:   cluster.ClusterConfig().Cloud.NetworkProvider,
		Provider:          cluster.ClusterConfig().Cloud.CloudProvider,
		ExternalProvider:  true, // Linode uses out-of-tree CCM
	}
	{
		td.KubeletExtraArgs = map[string]string{}
		for k, v := range cluster.ClusterConfig().KubeletExtraArgs {
			td.KubeletExtraArgs[k] = v
		}
		td.KubeletExtraArgs["node-labels"] = api.NodeLabels{
			api.NodePoolKey: machine.Name,
			api.RoleNodeKey: "",
		}.String()
		// ref: https://kubernetes.io/docs/admin/kubeadm/#cloud-provider-integrations-experimental
		td.KubeletExtraArgs["cloud-provider"] = "external" // --cloud-config is not needed
		td.KubeletExtraArgs["enable-controller-attach-detach"] = "false"
		td.KubeletExtraArgs["keep-terminated-pod-volumes"] = "true"
		if cluster.ClusterConfig().Cloud.CCMCredentialName == "" {
			panic(errors.New("no cloud controller manager credential found"))
		}

		cred, err := Store(ctx).Owner(owner).Credentials().Get(cluster.ClusterConfig().Cloud.CCMCredentialName)
		if err != nil {
			panic(err)
		}
		typed := credential.Linode{CommonSpec: credential.CommonSpec(cred.Spec)}
		if ok, err := typed.IsValid(); !ok {
			panic(err)
		}
		cloudConfig := &api.LinodeCloudConfig{
			Token: typed.APIToken(),
			Zone:  cluster.ClusterConfig().Cloud.Zone,
		}
		data, err := json.Marshal(cloudConfig)
		if err != nil {
			panic(err)
		}
		td.CloudConfig = string(data)
	}
	return td
}

func newMasterTemplateData(ctx context.Context, cluster *api.Cluster, machine *clusterv1.Machine, token, owner string) TemplateData {
	td := newNodeTemplateData(ctx, cluster, machine, token, owner)
	td.KubeletExtraArgs["node-labels"] = api.NodeLabels{
		api.NodePoolKey: machine.Name,
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
			BindPort: 6443,
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
		KubernetesVersion: cluster.ClusterConfig().KubernetesVersion,
		CertificatesDir:   "/etc/kubernetes/pki",
		APIServer: kubeadmapi.APIServer{
			ControlPlaneComponent: kubeadmapi.ControlPlaneComponent{
				ExtraArgs: cluster.Spec.Config.APIServerExtraArgs,
			},
			CertSANs: cluster.ClusterConfig().APIServerCertSANs,
		},
		ControllerManager: kubeadmapi.ControlPlaneComponent{
			ExtraArgs: cluster.ClusterConfig().ControllerManagerExtraArgs,
		},
		Scheduler: kubeadmapi.ControlPlaneComponent{
			ExtraArgs: cluster.ClusterConfig().SchedulerExtraArgs,
		},
		ClusterName: cluster.Name,
	}

	td.ControlPlaneEndpointsFromLB(&cfg, cluster)

	if token != "" {
		td.ControlPlaneJoin = true

		joinConf, err := td.JoinConfigurationYAML()
		if err != nil {
			panic(err)
		}
		td.JoinConfiguration = joinConf
	}

	td.ClusterConfiguration = &cfg

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
cmd='kubectl apply --kubeconfig /etc/kubernetes/admin.conf -f https://raw.githubusercontent.com/pharmer/addons/release-1.11/cloud-storage/rbac.yaml'
exec_until_success "$cmd"

#Deploy plugin
cmd='kubectl apply --kubeconfig /etc/kubernetes/admin.conf -f https://raw.githubusercontent.com/pharmer/addons/release-1.11/cloud-storage/{{ .Provider }}/flexplugin.yaml'
exec_until_success "$cmd"

#Deploy provisioner
cmd='kubectl apply --kubeconfig /etc/kubernetes/admin.conf -f https://raw.githubusercontent.com/pharmer/addons/release-1.11/cloud-storage/{{ .Provider }}/provisioner.yaml'
exec_until_success "$cmd"

{{ if not .IsVersionLessThan1_11}}
#Deploy initializer
cmd='kubectl apply --kubeconfig /etc/kubernetes/admin.conf -f https://raw.githubusercontent.com/pharmer/addons/release-1.11/cloud-controller-manager/initializer.yaml'
exec_until_success "$cmd"
{{ end }}
{{ end }}
`
)

func (conn *cloudConnector) renderStartupScript(cluster *api.Cluster, machine *clusterv1.Machine, token string, owner string) (string, error) {
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
		if err := tpl.ExecuteTemplate(&script, api.RoleMaster, newMasterTemplateData(conn.ctx, cluster, machine, token, owner)); err != nil {
			return "", err
		}
	} else {
		if err := tpl.ExecuteTemplate(&script, api.RoleNode, newNodeTemplateData(conn.ctx, cluster, machine, token, owner)); err != nil {
			return "", err
		}
	}
	return script.String(), nil
}

func (conn *cloudConnector) createStartupScript(cluster *api.Cluster) {

}
