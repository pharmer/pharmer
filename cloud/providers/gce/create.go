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
	api "pharmer.dev/pharmer/apis/v1alpha1"
	clusterapiGCE "pharmer.dev/pharmer/apis/v1alpha1/gce"
	proconfig "pharmer.dev/pharmer/apis/v1alpha1/gce"
	"pharmer.dev/pharmer/cloud/utils/kube"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) GetDefaultMachineProviderSpec(sku string, role api.MachineRole) (clusterapi.ProviderSpec, error) {
	cluster := cm.Cluster
	config := cluster.Spec.Config
	config.SSHUserName = cm.namer.AdminUsername()

	spec := clusterapiGCE.GCEMachineProviderSpec{
		TypeMeta: metav1.TypeMeta{
			APIVersion: proconfig.GCEProviderGroupName + "/" + proconfig.GCEProviderAPIVersion,
			Kind:       proconfig.GCEMachineProviderKind,
		},
		Zone:  config.Cloud.Zone,
		OS:    config.Cloud.InstanceImage,
		Roles: []api.MachineRole{role},
		Disks: []clusterapiGCE.Disk{
			{
				InitializeParams: clusterapiGCE.DiskInitializeParams{
					DiskType:   "pd-standard",
					DiskSizeGb: 30,
				},
			},
		},
		MachineType: sku,
	}

	rawSpec, err := clusterapiGCE.EncodeMachineSpec(&spec)
	if err != nil {
		return clusterapi.ProviderSpec{}, errors.Wrap(err, "Error encoding provider spec for gce cluster")
	}

	return clusterapi.ProviderSpec{
		Value:     rawSpec,
		ValueFrom: nil,
	}, nil
}

func (cm *ClusterManager) SetDefaultCluster() error {
	cluster := cm.Cluster

	n := namer{cluster: cluster}
	config := &cluster.Spec.Config

	config.Cloud.InstanceImageProject = "ubuntu-os-cloud"
	config.Cloud.InstanceImage = "ubuntu-1604-xenial-v20170721"
	config.Cloud.OS = "ubuntu-1604-lts"
	config.Cloud.GCE = &api.GoogleSpec{
		NetworkName: "default",
		NodeTags:    []string{n.NodePrefix()},
	}

	config.APIServerExtraArgs = make(map[string]string)
	config.APIServerExtraArgs["cloud-config"] = "/etc/kubernetes/ccm/cloud-config"

	config.KubeletExtraArgs["cloud-provider"] = cluster.ClusterConfig().Cloud.CloudProvider // requires --cloud-config

	cluster.Spec.Config.Cloud.Region = cluster.Spec.Config.Cloud.Zone[0 : len(cluster.Spec.Config.Cloud.Zone)-2]
	config.ControllerManagerExtraArgs = map[string]string{
		"cloud-config":   "/etc/kubernetes/ccm/cloud-config",
		"cloud-provider": config.Cloud.CloudProvider,
	}
	cluster.Spec.Config.APIServerExtraArgs = map[string]string{
		"cloud-provider": cluster.Spec.Config.Cloud.CloudProvider,
	}

	if cluster.Spec.ClusterAPI.ObjectMeta.Annotations == nil {
		cluster.Spec.ClusterAPI.ObjectMeta.Annotations = make(map[string]string)
	}

	// set clusterAPI provider-specs
	return clusterapiGCE.SetGCEclusterProviderConfig(&cluster.Spec.ClusterAPI, config.Cloud.Project, cm.Certs)
}

func (cm *ClusterManager) GetKubeConfig() (*api.KubeConfig, error) {
	return kube.GetAdminConfig(cm.StoreProvider.Certificates(cm.Cluster.Name), cm.Cluster, cm.GetCaCertPair())
}
