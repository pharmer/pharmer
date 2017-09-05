package api

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
	case *Instance:
		if u.APIVersion == "" {
			u.APIVersion = "v1alpha1"
		}
		u.Kind = "Instance"
		return nil
	}
	return errors.New("Unknown api object type")
}
