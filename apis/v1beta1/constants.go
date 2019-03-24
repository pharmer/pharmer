package v1beta1

import "time"

const (
	RoleMaster    = "master"
	RoleNode      = "node"
	RoleKeyPrefix = "node-role.kubernetes.io/"
	RoleMasterKey = RoleKeyPrefix + RoleMaster
	RoleNodeKey   = RoleKeyPrefix + RoleNode

	RoleLeader            = "leader"
	RoleMember            = "member"
	PharmerCluster        = "cluster.pharmer.io/cluster"
	KubeadmVersionKey     = "cluster.pharmer.io/kubeadm-version"
	NodePoolKey           = "cluster.pharmer.io/pool"
	KubeSystem_App        = "k8s-app"
	EtcdMemberKey         = "cluster.pharmer.io/etcd-type"
	EtcdServerAddress     = "cluster.pharmer.io/etcd-server-address"
	PharmerHASetup        = "cluster.pharmer.io/ha-setup"
	PharmerLoadBalancerIP = "cluster.pharmer.io/lb-ip"

	HostnameKey     = "kubernetes.io/hostname"
	ArchKey         = "beta.kubernetes.io/arch"
	InstanceTypeKey = "beta.kubernetes.io/instance-type"
	OSKey           = "beta.kubernetes.io/os"
	RegionKey       = "failure-domain.beta.kubernetes.io/region"
	ZoneKey         = "failure-domain.beta.kubernetes.io/zone"

	TokenDuration_10yr = 10 * 365 * 24 * time.Hour

	// ref: https://github.com/kubernetes/kubeadm/issues/629
	DeprecatedV19AdmissionControl = "NamespaceLifecycle,LimitRanger,ServiceAccount,PersistentVolumeLabel,DefaultStorageClass,ValidatingAdmissionWebhook,DefaultTolerationSeconds,MutatingAdmissionWebhook,ResourceQuota"
	DefaultV19AdmissionControl    = "NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,ValidatingAdmissionWebhook,DefaultTolerationSeconds,MutatingAdmissionWebhook,ResourceQuota"

	PodNetworkCalico  = "calico"
	PodNetworkFlannel = "flannel"
	PodNetworkCanal   = "canal"
)
