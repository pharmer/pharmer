package v1alpha1

import (
	"fmt"
	"strings"
)

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
