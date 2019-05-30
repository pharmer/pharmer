package gce

import (
	"bytes"

	"github.com/appscode/go/log"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	"gopkg.in/ini.v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func newNodeTemplateData(cm *CloudManager, cluster *api.Cluster, machine *clusterapi.Machine, token string) TemplateData {
	td := NewNodeTemplateData(cm, cluster, machine, token)
	td.ExternalProvider = false // GCE does not use out-of-tree CCM

	// ref: https://kubernetes.io/docs/admin/kubeadm/#cloud-provider-integrations-experimental
	td.KubeletExtraArgs["cloud-provider"] = cluster.ClusterConfig().Cloud.CloudProvider // requires --cloud-config
	// ref: https://github.com/kubernetes/kubernetes/blob/release-1.5/cluster/gce/configure-vm.sh#L846

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

	return td
}

func newMasterTemplateData(cm *CloudManager, cluster *api.Cluster, machine *clusterapi.Machine) TemplateData {
	td := newNodeTemplateData(cm, cluster, machine, "")
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

func (conn *cloudConnector) renderStartupScript(cluster *api.Cluster, machine *clusterapi.Machine, token string) (string, error) {
	return RenderStartupScript(machine, customTemplate, newMasterTemplateData(conn.CloudManager, cluster, machine), newNodeTemplateData(conn.CloudManager, cluster, machine, token))
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
