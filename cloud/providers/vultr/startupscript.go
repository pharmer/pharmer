package vultr

import (
	"bytes"
	"context"
	"encoding/json"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/credential"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/cert"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/pubkeypin"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/util"
)

func newNodeTemplateData(ctx context.Context, cluster *api.Cluster, machine *clusterv1.Machine, token string, owner string) TemplateData {
	cred, err := Store(ctx).Owner(owner).Credentials().Get(cluster.ClusterConfig().Cloud.CCMCredentialName)
	if err != nil {
		panic(err)
	}
	td := TemplateData{
		ClusterName:       cluster.Name,
		KubernetesVersion: machine.Spec.Versions.ControlPlane,
		KubeadmToken:      token,
		CloudCredential:   cred.Spec.Data,
		CAHash:            pubkeypin.Hash(CACert(ctx)),
		CAKey:             string(cert.EncodePrivateKeyPEM(CAKey(ctx))),
		FrontProxyKey:     string(cert.EncodePrivateKeyPEM(FrontProxyCAKey(ctx))),
		APIServerAddress:  cluster.APIServerAddress(),
		NetworkProvider:   cluster.ClusterConfig().Cloud.NetworkProvider,
		Provider:          cluster.ClusterConfig().Cloud.CloudProvider,
		ExternalProvider:  true, // Vultr uses out-of-tree CCM
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
		if cluster.ClusterConfig().Cloud.CCMCredentialName == "" {
			panic(errors.New("no cloud controller manager credential found"))
		}
		cred, err := Store(ctx).Owner(owner).Credentials().Get(cluster.ClusterConfig().CredentialName)
		if err != nil {
			panic(err)
		}
		typed := credential.DigitalOcean{CommonSpec: credential.CommonSpec(cred.Spec)}
		if ok, err := typed.IsValid(); !ok {
			panic(err)
		}
		cloudConfig := &api.VultrCloudConfig{
			Token: typed.Token(),
		}
		data, err := json.Marshal(cloudConfig)
		if err != nil {
			panic(err)
		}
		td.CloudConfig = string(data)
	}
	if machine.Spec.Versions.ControlPlane == "" {
		td.KubernetesVersion = machine.Spec.Versions.Kubelet
	}
	return td
}

func newMasterTemplateData(ctx context.Context, cluster *api.Cluster, machine *clusterv1.Machine, owner string) TemplateData {
	td := newNodeTemplateData(ctx, cluster, machine, "", owner)
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
		// "external": cloudprovider not supported for apiserver and controller-manager
		// https://github.com/kubernetes/kubernetes/pull/50545
		APIServer: kubeadmapi.APIServer{
			ControlPlaneComponent: kubeadmapi.ControlPlaneComponent{
				ExtraArgs: cluster.ClusterConfig().APIServerExtraArgs,
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
	td.ClusterConfiguration = &cfg
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

{{ define "prepare-host" }}
# https://www.vultr.com/docs/configuring-private-network
PRIVATE_ADDRESS=$(/usr/bin/curl -fsSL --retry 5 http://169.254.169.254/v1/interfaces/1/ipv4/address 2> /dev/null)
PRIVATE_NETMASK=$(/usr/bin/curl -fsSL --retry 5 http://169.254.169.254/v1/interfaces/1/ipv4/netmask 2> /dev/null)
/bin/cat >>/etc/network/interfaces <<EOF

auto ens7
iface ens7 inet static
    address $PRIVATE_ADDRESS
    netmask $PRIVATE_NETMASK
    mtu 1450
EOF
ifup ens7
{{ end }}
`

// INSTANCE_ID=$(/usr/bin/curl -fsSL --retry 5 http://169.254.169.254/v1/instanceid 2> /dev/null)
// PRIVATE_ADDRESS=$(pre-k vultr private-ip --token={{ index .CloudCredential "token" }} --instance-id=$INSTANCE_ID)
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
		if err := tpl.ExecuteTemplate(&script, api.RoleMaster, newMasterTemplateData(conn.ctx, conn.cluster, machine, owner)); err != nil {
			return "", err
		}
	} else {
		if err := tpl.ExecuteTemplate(&script, api.RoleNode, newNodeTemplateData(conn.ctx, conn.cluster, machine, token, owner)); err != nil {
			return "", err
		}
	}
	return script.String(), nil
}

func (conn *cloudConnector) createStartupScript(cluster *api.Cluster) {

}
