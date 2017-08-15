package api

const (
	CertTrusted = iota - 1
	CertRoot
	CertNSRoot
	CertIntermediate
	CertLeaf

	RoleKubernetesMaster = "kubernetes-master"
	RoleKubernetesPool   = "kubernetes-pool"

	CIBotUser      = "ci-bot"
	ClusterBotUser = "k8s-bot"
)

const (
	JobStatus_Requested = "REQUESTED"
	JobStatus_Running   = "RUNNING"
	JobStatus_Done      = "DONE"
	JobStatus_Failed    = "FAILED"
)

/*
+---------------------------------+
|                                 |
|  +---------+     +---------+    |     +--------+
|  | PENDING +-----> FAILING +----------> FAILED |
|  +----+----+     +---------+    |     +--------+
|       |                         |
|       |                         |
|  +----v----+                    |
|  |  READY  |                    |
|  +----+----+                    |
|       |                         |
|       |                         |
|  +----v-----+                   |
|  | DELETING |                   |
|  +----+-----+                   |
|       |                         |
+---------------------------------+
        |
        |
   +----v----+
   | DELETED |
   +---------+
*/
const (
	KubernetesStatus_Pending  = "PENDING"
	KubernetesStatus_Failing  = "FAILING"
	KubernetesStatus_Failed   = "FAILED"
	KubernetesStatus_Ready    = "READY"
	KubernetesStatus_Deleting = "DELETING"
	KubernetesStatus_Deleted  = "DELETED"

	// ref: https://github.com/liggitt/kubernetes.github.io/blob/1d14da9c42266801c9ac13cb9608b9f8010dda49/docs/admin/authorization/rbac.md#default-clusterroles-and-clusterrolebindings
	KubernetesAccessModeGroupTeamAdmin    = "kubernetes:team-admin"
	KubernetesAccessModeGroupClusterAdmin = "kubernetes:cluster-admin"
	KubernetesAccessModeGroupAdmin        = "kubernetes:admin"
	KubernetesAccessModeGroupEditor       = "kubernetes:editor"
	KubernetesAccessModeGroupViewer       = "kubernetes:viewer"
	KubernetesAccessModeGroupDenyAccess   = "deny-access"
)

const (
	KubernetesInstanceStatus_Ready   = "READY"
	KubernetesInstanceStatus_Deleted = "DELETED"
)
