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
package v1alpha1

import (
	kubeadmconst "k8s.io/kubernetes/cmd/kubeadm/app/constants"
)

const (
	RoleMaster    = "master"
	RoleNode      = "node"
	RoleKeyPrefix = "node-role.kubernetes.io/"
	RoleMasterKey = RoleKeyPrefix + RoleMaster
	RoleNodeKey   = RoleKeyPrefix + RoleNode

	PharmerCluster    = "cluster.pharmer.io/cluster"
	KubeadmVersionKey = "cluster.pharmer.io/kubeadm-version"
	NodePoolKey       = "cluster.pharmer.io/pool"

	PodNetworkCalico  = "calico"
	PodNetworkFlannel = "flannel"
	PodNetworkCanal   = "canal"

	CACertName                 = kubeadmconst.CACertAndKeyBaseName
	CACertCommonName           = kubeadmconst.CACertAndKeyBaseName
	FrontProxyCACertName       = kubeadmconst.FrontProxyCACertAndKeyBaseName
	FrontProxyCACertCommonName = kubeadmconst.FrontProxyCACertAndKeyBaseName
	SAKeyName                  = kubeadmconst.ServiceAccountKeyBaseName
	SAKeyCommonName            = kubeadmconst.ServiceAccountKeyBaseName
	ETCDCACertName             = kubeadmconst.EtcdCACertAndKeyBaseName
	ETCDCACertCommonName       = "kubernetes"
)
