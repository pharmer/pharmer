package api

import "encoding/json"

type ClustersInfo struct {
	Name    string                 `json:"name"`
	Cluster map[string]interface{} `json:"cluster"`
}

type UserInfo struct {
	Name string                 `json:"name"`
	User map[string]interface{} `json:"user"`
}

type ContextInfo struct {
	Name     string                 `json:"name"`
	Contextt map[string]interface{} `json:"context"`
}

// Adapted from https://github.com/kubernetes/client-go/blob/master/tools/clientcmd/api/v1/types.go#L27
// Simplified to avoid dependency on client-go
type KubeConfig struct {
	Kind           string                 `json:"kind,omitempty"`
	APIVersion     string                 `json:"apiVersion,omitempty"`
	Clusters       []*ClustersInfo        `json:"clusters"`
	Contexts       []*ContextInfo         `json:"contexts"`
	CurrentContext string                 `json:"current-context"`
	Preferences    map[string]interface{} `json:"preferences"`
	Users          []*UserInfo            `json:"users"`
	Extensions     json.RawMessage        `json:"extensions,omitempty"`
}
