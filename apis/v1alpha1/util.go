package v1alpha1

import (
	"errors"
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
	case *NodeGroup:
		if u.APIVersion == "" {
			u.APIVersion = "v1alpha1"
		}
		u.Kind = "NodeGroup"
		return nil
	case *Node:
		if u.APIVersion == "" {
			u.APIVersion = "v1alpha1"
		}
		u.Kind = "Instance"
		return nil
	}
	return errors.New("Unknown api object type")
}
