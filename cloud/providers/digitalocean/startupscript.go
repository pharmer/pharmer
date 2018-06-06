package digitalocean

import (
	"bytes"
	"context"
	"fmt"

	api "github.com/pharmer/pharmer/apis/v1"
	. "github.com/pharmer/pharmer/cloud"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/cert"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1alpha1"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/pubkeypin"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func newNodeTemplateData(ctx context.Context, cluster *api.Cluster, machine *clusterv1.Machine, token string) TemplateData {
	clusterConfig := cluster.ProviderConfig()
	td := TemplateData{
		ClusterName:       cluster.Name,
		KubernetesVersion: clusterConfig.KubernetesVersion,
		KubeadmToken:      token,
		CAHash:            pubkeypin.Hash(CACert(ctx)),
		CAKey:             string(cert.EncodePrivateKeyPEM(CAKey(ctx))),
		SAKey:             string(cert.EncodePrivateKeyPEM(SaKey(ctx))),
		FrontProxyKey:     string(cert.EncodePrivateKeyPEM(FrontProxyCAKey(ctx))),
		ETCDCAKey:         string(cert.EncodePrivateKeyPEM(EtcdCaKey(ctx))),
		ETCDTLSEnabled:    false,
		APIServerAddress:  cluster.APIServerAddress(),
		NetworkProvider:   clusterConfig.Cloud.NetworkProvider,
		Provider:          clusterConfig.Cloud.CloudProvider,
		ExternalProvider:  true, // DigitalOcean uses out-of-tree CCM
	}
	{
		td.KubeletExtraArgs = map[string]string{}
		for k, v := range clusterConfig.KubeletExtraArgs {
			td.KubeletExtraArgs[k] = v
		}
		fmt.Println(machine)
		machineConfig, err := cluster.MachineProviderConfig(machine)
		if err != nil {
			panic(err)
		}
		for k, v := range machineConfig.Config.KubeletExtraArgs {
			td.KubeletExtraArgs[k] = v
		}
		td.KubeletExtraArgs["node-labels"] = api.NodeLabels{
			api.NodePoolKey: machine.Name,
			api.RoleNodeKey: "",
		}.String()
		// ref: https://kubernetes.io/docs/admin/kubeadm/#cloud-provider-integrations-experimental
		td.KubeletExtraArgs["cloud-provider"] = "external" // --cloud-config is not needed
		td.KubeletExtraArgs["enable-controller-attach-detach"] = "false"
	}
	return td
}

func newMasterTemplateData(ctx context.Context, cluster *api.Cluster, machine *clusterv1.Machine) TemplateData {
	clusterConfig := cluster.ProviderConfig()
	td := newNodeTemplateData(ctx, cluster, machine, "")
	td.KubeletExtraArgs["node-labels"] = api.NodeLabels{
		api.NodePoolKey: machine.Name,
	}.String()

	if machine.Labels[api.EtcdMemberKey] == api.RoleMember {
		//extraArgs["server-address"] = machine.Labels[api.EtcdServerAddress]
		td.ETCDServerAddress = machine.Labels[api.EtcdServerAddress] //fmt.Sprintf("http://%s:2379", machine.Labels[api.EtcdServerAddress])
	}

	cfg := kubeadmapi.MasterConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubeadm.k8s.io/v1alpha1",
			Kind:       "MasterConfiguration",
		},
		API: kubeadmapi.API{
			AdvertiseAddress: clusterConfig.API.AdvertiseAddress,
			BindPort:         clusterConfig.API.BindPort,
		},
		Networking: kubeadmapi.Networking{
			ServiceSubnet: cluster.Spec.ClusterAPI.Spec.ClusterNetwork.Services.CIDRBlocks[0],
			PodSubnet:     cluster.Spec.ClusterAPI.Spec.ClusterNetwork.Pods.CIDRBlocks[0],
			DNSDomain:     cluster.Spec.ClusterAPI.Spec.ClusterNetwork.ServiceDomain,
		},
		KubernetesVersion: clusterConfig.KubernetesVersion,
		Etcd: kubeadmapi.Etcd{
			Image: EtcdImage,
			//ExtraArgs: extraArgs,
		},
		CertificatesDir: "/etc/kubernetes/pki",
		// "external": cloudprovider not supported for apiserver and controller-manager
		// https://github.com/kubernetes/kubernetes/pull/50545
		CloudProvider:              "",
		APIServerExtraArgs:         clusterConfig.APIServerExtraArgs,
		ControllerManagerExtraArgs: clusterConfig.ControllerManagerExtraArgs,
		SchedulerExtraArgs:         clusterConfig.SchedulerExtraArgs,
		APIServerCertSANs:          clusterConfig.APIServerCertSANs,
		NodeName:                   machine.Name,
	}
	if _, found := machine.Labels[api.PharmerHASetup]; found {
		td.HASetup = true
		td.LoadBalancerIp = machine.Labels[api.PharmerLoadBalancerIP]
		cfg.APIServerCertSANs = append(cfg.APIServerCertSANs, machine.Labels[api.PharmerLoadBalancerIP])
	}

	if clusterConfig.IsMinorVersion("1.10") {
		td.ETCDTLSEnabled = true
	}

	td.MasterConfiguration = &cfg
	return td
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
cmd='kubectl apply --kubeconfig /etc/kubernetes/admin.conf -f https://raw.githubusercontent.com/pharmer/addons/release-1.10/cloud-storage/rbac.yaml'
exec_until_success "$cmd"

#Deploy plugin
cmd='kubectl apply --kubeconfig /etc/kubernetes/admin.conf -f https://raw.githubusercontent.com/pharmer/addons/release-1.10/cloud-storage/{{ .Provider }}/flexplugin.yaml'
exec_until_success "$cmd"

#Deploy provisioner
cmd='kubectl apply --kubeconfig /etc/kubernetes/admin.conf -f https://raw.githubusercontent.com/pharmer/addons/release-1.10/cloud-storage/{{ .Provider }}/provisioner.yaml'
exec_until_success "$cmd"
{{ end }}

{{ define "prepare-host" }}
NODE_NAME=$(hostname)
{{ end }}
`
)

func (conn *cloudConnector) renderStartupScript(cluster *api.Cluster, machine *clusterv1.Machine, token string) (string, error) {
	tpl, err := StartupScriptTemplate.Clone()
	if err != nil {
		return "", err
	}
	tpl, err = tpl.Parse(customTemplate)
	if err != nil {
		return "", err
	}

	var script bytes.Buffer
	if IsMaster(machine) {
		if err := tpl.ExecuteTemplate(&script, api.RoleMaster, newMasterTemplateData(conn.ctx, conn.cluster, machine)); err != nil {
			return "", err
		}
	} else {
		if err := tpl.ExecuteTemplate(&script, api.RoleNode, newNodeTemplateData(conn.ctx, conn.cluster, machine, token)); err != nil {
			return "", err
		}
	}
	return script.String(), nil
}
