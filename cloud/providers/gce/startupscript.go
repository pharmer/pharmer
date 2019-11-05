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
package gce

import (
	"bytes"

	api "pharmer.dev/pharmer/apis/v1alpha1"
	"pharmer.dev/pharmer/cloud"

	"github.com/appscode/go/log"
	"gopkg.in/ini.v1"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) NewNodeTemplateData(machine *clusterapi.Machine, token string, td cloud.TemplateData) cloud.TemplateData {
	td.ExternalProvider = false // GCE does not use out-of-tree CCM

	n := namer{cm.Cluster}

	cloudConfig := &api.GCECloudConfig{
		ProjectID:          cm.Cluster.Spec.Config.Cloud.Project,
		NetworkName:        cm.Cluster.Spec.Config.Cloud.GCE.NetworkName,
		NodeTags:           cm.Cluster.Spec.Config.Cloud.GCE.NodeTags,
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
		// TODO: should we handle error in better way?
		panic(err)
	}
	td.CloudConfig = buf.String()

	return td
}

func (cm *ClusterManager) NewMasterTemplateData(machine *clusterapi.Machine, token string, td cloud.TemplateData) cloud.TemplateData {
	hostPath := kubeadmapi.HostPathMount{
		Name:      "cloud-config",
		HostPath:  "/etc/kubernetes/ccm",
		MountPath: "/etc/kubernetes/ccm",
	}

	cfg := cloud.GetDefaultKubeadmClusterConfig(cm.Cluster, &hostPath)
	td.ClusterConfiguration = cfg

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
