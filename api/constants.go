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
	JobPhaseRequested = "REQUESTED"
	JobPhaseRunning   = "RUNNING"
	JobPhaseDone      = "DONE"
	JobPhaseFailed    = "FAILED"
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
	ClusterPhasePending  = "PENDING"
	ClusterPhaseFailing  = "FAILING"
	ClusterPhaseFailed   = "FAILED"
	ClusterPhaseReady    = "READY"
	ClusterPhaseDeleting = "DELETING"
	ClusterPhaseDeleted  = "DELETED"

	// ref: https://github.com/liggitt/kubernetes.github.io/blob/1d14da9c42266801c9ac13cb9608b9f8010dda49/docs/admin/authorization/rbac.md#default-clusterroles-and-clusterrolebindings
	KubernetesAccessModeGroupTeamAdmin    = "kubernetes:team-admin"
	KubernetesAccessModeGroupClusterAdmin = "kubernetes:cluster-admin"
	KubernetesAccessModeGroupAdmin        = "kubernetes:admin"
	KubernetesAccessModeGroupEditor       = "kubernetes:editor"
	KubernetesAccessModeGroupViewer       = "kubernetes:viewer"
	KubernetesAccessModeGroupDenyAccess   = "deny-access"
)

const (
	InstancePhaseReady   = "READY"
	InstancePhaseDeleted = "DELETED"
)
