package v1beta1

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
