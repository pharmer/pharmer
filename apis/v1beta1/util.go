package v1beta1

import (
	"github.com/pkg/errors"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func AssignTypeKind(v interface{}) error {
	switch u := v.(type) {
	case *PharmerConfig:
		if u.APIVersion == "" {
			u.APIVersion = "v1"
		}
		u.Kind = "PharmerConfig"
		return nil
	case *Cluster:
		if u.APIVersion == "" {
			u.APIVersion = "v1"
		}
		u.Kind = "Cluster"
		return nil
	case *Credential:
		if u.APIVersion == "" {
			u.APIVersion = "v1"
		}
		u.Kind = "Credential"
		return nil
	case *clusterapi.Cluster:
		if u.APIVersion == "" {
			u.APIVersion = "cluster.k8s.io/v1alpha1"
		}
		u.Kind = "Cluster"
		return nil
	case *clusterapi.Machine:
		if u.APIVersion == "" {
			u.APIVersion = "cluster.k8s.io/v1alpha1"
		}
		u.Kind = "Machine"
		return nil
	case *clusterapi.MachineSet:
		if u.APIVersion == "" {
			u.APIVersion = "cluster.k8s.io/v1alpha1"
		}
		u.Kind = "MachineSet"
		return nil
	}
	return errors.New("Unknown api object type")
}
