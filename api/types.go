package api

import (
	"errors"
)

const (
	RoleKubernetesMaster = "kubernetes-master"
	RoleKubernetesPool   = "kubernetes-pool"
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

// ClusterPhase is a label for the condition of a Cluster at the current time.
type ClusterPhase string

// These are the valid statuses of Cluster.
const (
	ClusterPending  ClusterPhase = "Pending"
	ClusterFailing  ClusterPhase = "Failing"
	ClusterFailed   ClusterPhase = "Failed"
	ClusterReady    ClusterPhase = "Ready"
	ClusterDeleting ClusterPhase = "Deleting"
	ClusterDeleted  ClusterPhase = "Deleted"
)

// InstancePhase is a label for the condition of an Instance at the current time.
type InstancePhase string

const (
	InstanceReady   InstancePhase = "Ready"
	InstanceDeleted InstancePhase = "Deleted"
)

func AssignTypeKind(v interface{}) error {
	switch u := v.(type) {
	case *PharmerConfig:
		if u.APIVersion == "" {
			u.APIVersion = "v1alpha1"
		}
		u.Kind = "PharmerConfig"
		return nil
	case *Cluster:
		if u.APIVersion == "" {
			u.APIVersion = "v1alpha1"
		}
		u.Kind = "Cluster"
		return nil
	case *Credential:
		if u.APIVersion == "" {
			u.APIVersion = "v1alpha1"
		}
		u.Kind = "Credential"
		return nil
	case *InstanceGroup:
		if u.APIVersion == "" {
			u.APIVersion = "v1alpha1"
		}
		u.Kind = "InstanceGroup"
		return nil
	case *Instance:
		if u.APIVersion == "" {
			u.APIVersion = "v1alpha1"
		}
		u.Kind = "Instance"
		return nil
	}
	return errors.New("Unknown api object type")
}
