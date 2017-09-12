package util

import (
	"fmt"
	"github.com/appscode/pharmer/api"
	"strings"
)

func GetSupportedResource(resource string) (string, error) {
	switch strings.ToLower(resource) {
	case strings.ToLower(api.ResourceTypeCluster),
		strings.ToLower(api.ResourceNameCluster),
		strings.ToLower(api.ResourceKindCluster):
		return api.ResourceTypeCluster, nil
	case strings.ToLower(api.ResourceTypeNodeGroup),
		strings.ToLower(api.ResourceNameNodeGroup),
		strings.ToLower(api.ResourceKindNodeGroup),
		strings.ToLower(api.ResourceCodeNodeGroup):
		return api.ResourceTypeNodeGroup, nil
	default:
		return "", fmt.Errorf(`pharmer doesn't support a resource type "%v"`, resource)
	}
}

func GetAllSupportedResources() []string {
	return []string{api.ResourceTypeCluster, api.ResourceTypeNodeGroup}
}
