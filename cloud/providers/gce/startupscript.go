package gce

import (
	"bytes"
	"context"

	"github.com/appscode/go/log"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	"gomodules.xyz/cert"
	ini "gopkg.in/ini.v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/pubkeypin"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/util"
)

func newNodeTemplateData(ctx context.Context, cluster *api.Cluster, machine *clusterv1.Machine, token string) TemplateData {
	td := TemplateData{
		ClusterName:       cluster.Name,
		KubernetesVersion: machine.Spec.Versions.ControlPlane,
		KubeadmToken:      token,
		CAHash:            pubkeypin.Hash(CACert(ctx)),
		CAKey:             string(cert.EncodePrivateKeyPEM(CAKey(ctx))),
		SAKey:             string(cert.EncodePrivateKeyPEM(SaKey(ctx))),
		FrontProxyKey:     string(cert.EncodePrivateKeyPEM(FrontProxyCAKey(ctx))),
		ETCDCAKey:         string(cert.EncodePrivateKeyPEM(EtcdCaKey(ctx))),
		APIServerAddress:  cluster.APIServerAddress(),
		NetworkProvider:   cluster.ClusterConfig().Cloud.NetworkProvider,
		Provider:          cluster.ClusterConfig().Cloud.CloudProvider,
		ExternalProvider:  false, // GCE does not use out-of-tree CCM
	}

	{
		td.KubeletExtraArgs = map[string]string{}
		for k, v := range cluster.Spec.Config.KubeletExtraArgs {
			td.KubeletExtraArgs[k] = v
		}

		td.KubeletExtraArgs["node-labels"] = api.NodeLabels{
			api.NodePoolKey: machine.Name,
			api.RoleNodeKey: "",
		}.String()
		// ref: https://kubernetes.io/docs/admin/kubeadm/#cloud-provider-integrations-experimental
		td.KubeletExtraArgs["cloud-provider"] = cluster.ClusterConfig().Cloud.CloudProvider // requires --cloud-config
		// ref: https://github.com/kubernetes/kubernetes/blob/release-1.5/cluster/gce/configure-vm.sh#L846

		if cluster.ClusterConfig().Cloud.CCMCredentialName == "" {
			panic(errors.New("no cloud controller manager credential found"))
		}

		n := namer{cluster}

		cloudConfig := &api.GCECloudConfig{
			ProjectID:          cluster.Spec.Config.Cloud.Project,
			NetworkName:        cluster.Spec.Config.Cloud.GCE.NetworkName,
			NodeTags:           cluster.Spec.Config.Cloud.GCE.NodeTags,
			NodeInstancePrefix: n.NodePrefix(),
			Multizone:          false,
		}

		cfg := ini.Empty()
		err := cfg.Section("global").ReflectFrom(cloudConfig)
		if err != nil {
			log.Info(err)
		}
		if err != nil {
			panic(err)
		}
		var buf bytes.Buffer
		_, err = cfg.WriteTo(&buf)
		if err != nil {
			panic(err)
		}
		td.CloudConfig = buf.String()

		// ref: https://github.com/kubernetes/kubernetes/blob/1910086bbce4f08c2b3ab0a4c0a65c913d4ec921/cmd/kubeadm/app/phases/controlplane/manifests.go#L41
		// Kubeadm will send cloud-config to kube-apiserver and kube-controller-manager
		// ref: https://github.com/kubernetes/kubernetes/blob/1910086bbce4f08c2b3ab0a4c0a65c913d4ec921/cmd/kubeadm/app/phases/controlplane/manifests.go#L193
		// ref: https://github.com/kubernetes/kubernetes/blob/1910086bbce4f08c2b3ab0a4c0a65c913d4ec921/cmd/kubeadm/app/phases/controlplane/manifests.go#L230
	}
	return td
}

func newMasterTemplateData(ctx context.Context, cluster *api.Cluster, machine *clusterv1.Machine) TemplateData {
	td := newNodeTemplateData(ctx, cluster, machine, "")
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
			//AdvertiseAddress: cluster.Spec.Config.API.AdvertiseAddress,
			//BindPort:         cluster.Spec.Config.API.BindPort,
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
			ExtraArgs:    cluster.Spec.Config.ControllerManagerExtraArgs,
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
{{ define "init-os" }}
# We rely on DNS for a lot, and it's just not worth doing a whole lot of startup work if this isn't ready yet.
# ref: https://github.com/kubernetes/kubernetes/blob/443908193d564736d02efdca4c9ba25caf1e96fb/cluster/gce/configure-vm.sh#L24
ensure_basic_networking() {
  until getent hosts metadata.google.internal &>/dev/null; do
    echo 'Waiting for functional DNS (trying to resolve metadata.google.internal)...'
    sleep 3
  done
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

{{ define "mount-master-pd" }}
pre-k mount-master-pd --provider=gce
{{ end }}

{{ define "cloud-config" }}
mkdir -p /etc/kubernetes/ccm
cat > /etc/kubernetes/ccm/cloud-config <<EOF
{{ .CloudConfig }}
EOF
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
	if util.IsControlPlaneMachine(machine) {
		if err := tpl.ExecuteTemplate(&script, api.RoleMaster, newMasterTemplateData(conn.ctx, cluster, machine)); err != nil {
			return "", err
		}
	} else {
		if err := tpl.ExecuteTemplate(&script, api.RoleNode, newNodeTemplateData(conn.ctx, cluster, machine, token)); err != nil {
			return "", err
		}
	}
	return script.String(), nil
}
