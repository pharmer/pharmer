package util

import (
	"fmt"
	"strings"
)

func GetSupportedResource(resource string) (string, error) {
	switch strings.ToLower(resource) {
	case strings.ToLower("cluster"):
		return "*api.Cluster", nil
	default:
		return "", fmt.Errorf(`pharmer doesn't support a resource type "%v"`, resource)
	}
}

func GetAllSupportedResources() []string {
	return []string{"*api.Cluster"}
}
