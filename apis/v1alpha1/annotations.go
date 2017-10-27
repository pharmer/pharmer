package v1alpha1

const (
	RoleMaster    = "master"
	RoleNode      = "node"
	RoleKeyPrefix = "node-role.kubernetes.io/"
	RoleMasterKey = RoleKeyPrefix + RoleMaster
	RoleNodeKey   = RoleKeyPrefix + RoleNode

	KubeadmVersionKey = "cloud.appscode.com/kubeadm-version"
	NodePoolKey       = "cloud.appscode.com/pool"
	KubeSystem_App    = "k8s-app"

	HostnameKey     = "kubernetes.io/hostname"
	ArchKey         = "beta.kubernetes.io/arch"
	InstanceTypeKey = "beta.kubernetes.io/instance-type"
	OSKey           = "beta.kubernetes.io/os"
	RegionKey       = "failure-domain.beta.kubernetes.io/region"
	ZoneKey         = "failure-domain.beta.kubernetes.io/zone"
)
