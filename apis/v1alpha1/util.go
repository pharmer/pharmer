package v1alpha1

import (
	"errors"
	"fmt"
	"strings"
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

func GetSupportedResource(resource string) (string, error) {
	switch strings.ToLower(resource) {
	case strings.ToLower(ResourceTypeCluster),
		strings.ToLower(ResourceNameCluster),
		strings.ToLower(ResourceKindCluster):
		return ResourceTypeCluster, nil
	case strings.ToLower(ResourceTypeNodeGroup),
		strings.ToLower(ResourceNameNodeGroup),
		strings.ToLower(ResourceKindNodeGroup),
		strings.ToLower(ResourceCodeNodeGroup):
		return ResourceTypeNodeGroup, nil
	default:
		return "", fmt.Errorf(`pharmer doesn't support a resource type "%v"`, resource)
	}
}

func GetAllSupportedResources() []string {
	return []string{ResourceTypeCluster, ResourceTypeNodeGroup}
}
