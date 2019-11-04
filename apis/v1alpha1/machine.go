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
)

const (
	ResourceCodeNodeGroup = "ng"
	ResourceKindNodeGroup = "NodeGroup"
	ResourceNameNodeGroup = "nodegroup"
	ResourceTypeNodeGroup = "nodegroups"
)

type IPType string

type NodeType string

const (
	NodeTypeSpot NodeType = "spot"

	MachineSlecetor = "cluster.pharmer.io/mg"
)

type MachineRole string

const (
	MasterMachineRole MachineRole = "Master"
	NodeMachineRole   MachineRole = "Node"
)

type NodeSpec struct {
	SKU              string            `json:"sku,omitempty"`
	DiskType         string            `json:"nodeDiskType,omitempty"`
	DiskSize         int64             `json:"nodeDiskSize,omitempty"`
	ExternalIPType   IPType            `json:"externalIPType,omitempty"`
	KubeletExtraArgs map[string]string `json:"kubeletExtraArgs,omitempty"`
	Type             NodeType          `json:"type,omitempty"`
	SpotPriceMax     float64           `json:"spotPriceMax,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type NodeInfo struct {
	metav1.TypeMeta `json:",inline"`
	Name            string `json:"name,omitempty"`
	ExternalID      string `json:"externalID,omitempty"`
	PublicIP        string `json:"publicIP,omitempty"`
	PrivateIP       string `json:"privateIP,omitempty"`
	DiskId          string `json:"diskID,omitempty"`
}
