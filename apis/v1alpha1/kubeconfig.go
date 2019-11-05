/*
Copyright The Pharmer Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// Config holds the information needed to build connect to remote kubernetes clusters as a given user
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type KubeConfig struct {
	metav1.TypeMeta `json:",inline"`
	// Preferences holds general information to be use for cli interactions
	Preferences Preferences `json:"preferences"`
	// Clusters is a map of referencable names to cluster configs
	Cluster NamedCluster `json:"cluster"`
	// AuthInfos is a map of referencable names to user configs
	AuthInfo NamedAuthInfo `json:"user"`
	// Contexts is a map of referencable names to context configs
	Context NamedContext `json:"context"`
}

type Preferences struct {
	// +optional
	Colors bool `json:"colors,omitempty"`
}

// NamedCluster relates nicknames to cluster information
type NamedCluster struct {
	// Name is the nickname for this Cluster
	Name string `json:"name"`
	// Server is the address of the kubernetes cluster (https://hostname:port).
	Server string `json:"server"`
	// CertificateAuthorityData contains PEM-encoded CertKeyPair authority certificates. Overrides CertificateAuthorityData
	// +optional
	CertificateAuthorityData []byte `json:"certificateAuthorityData,omitempty"`
}

// NamedContext relates nicknames to context information
type NamedContext struct {
	// Name is the nickname for this Context
	Name string `json:"name"`
	// Cluster is the name of the cluster for this context
	Cluster string `json:"cluster"`
	// AuthInfo is the name of the authInfo for this context
	AuthInfo string `json:"user"`
}

// NamedAuthInfo relates nicknames to auth information
type NamedAuthInfo struct {
	// Name is the nickname for this AuthInfo
	Name string `json:"name"`
	// ClientCertificateData contains PEM-encoded data from a client cert file for TLS.
	// +optional
	ClientCertificateData []byte `json:"clientCertificateData,omitempty"`
	// ClientKeyData contains PEM-encoded data from a client key file for TLS.
	// +optional
	ClientKeyData []byte `json:"clientKeyData,omitempty"`
	// Token is the bearer token for authentication to the kubernetes cluster.
	// +optional
	Token string `json:"token,omitempty"`
	// Username is the username for basic authentication to the kubernetes cluster.
	// +optional
	Username string `json:"username,omitempty"`
	// Password is the password for basic authentication to the kubernetes cluster.
	// +optional
	Password string `json:"password,omitempty"`
	// +optional
	Exec *ExecConfig `json:"exec,omitempty"`
}

// ExecConfig specifies a command to provide client credentials. The command is exec'd
// and outputs structured stdout holding credentials.
//
// See the client.authentiction.k8s.io API group for specifications of the exact input
// and output format
type ExecConfig struct {
	// Command to execute.
	Command string `json:"command"`
	// Arguments to pass to the command when executing it.
	// +optional
	Args []string `json:"args"`
	// Env defines additional environment variables to expose to the process. These
	// are unioned with the host's environment, as well as variables client-go uses
	// to pass argument to the plugin.
	// +optional
	Env []ExecEnvVar `json:"env"`

	// Preferred input version of the ExecInfo. The returned ExecCredentials MUST use
	// the same encoding version as the input.
	APIVersion string `json:"apiVersion,omitempty"`
}

// ExecEnvVar is used for setting environment variables when executing an exec-based
// credential plugin.
type ExecEnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func Convert_KubeConfig_To_Config(in *KubeConfig) *clientcmdapi.Config {
	return &clientcmdapi.Config{
		Kind:       "Config",
		APIVersion: clientcmdapi.SchemeGroupVersion.String(),
		Preferences: clientcmdapi.Preferences{
			Colors: in.Preferences.Colors,
		},
		Clusters: map[string]*clientcmdapi.Cluster{
			in.Cluster.Name: {
				Server:                   in.Cluster.Server,
				CertificateAuthorityData: append([]byte(nil), in.Cluster.CertificateAuthorityData...),
			},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			in.AuthInfo.Name: {
				Token:                 in.AuthInfo.Token,
				ClientCertificateData: append([]byte(nil), in.AuthInfo.ClientCertificateData...),
				ClientKeyData:         append([]byte(nil), in.AuthInfo.ClientKeyData...),
			},
		},
		Contexts: map[string]*clientcmdapi.Context{
			in.Context.Name: {
				Cluster:  in.Context.Cluster,
				AuthInfo: in.Context.AuthInfo,
			},
		},
		CurrentContext: in.Context.Name,
	}
}
