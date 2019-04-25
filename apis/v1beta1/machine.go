package v1beta1

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

const (
	ResourceCodeNodeGroup = "ng"
	ResourceKindNodeGroup = "NodeGroup"
	ResourceNameNodeGroup = "nodegroup"
	ResourceTypeNodeGroup = "nodegroups"
)

type IPType string

const (
	IPTypeEphemeral IPType = "Ephemeral"
	IPTypeReserved  IPType = "Reserved"
)

type NodeType string

const (
	NodeTypeRegular NodeType = "regular"
	NodeTypeSpot    NodeType = "spot"

	MachineSlecetor = "cluster.pharmer.io/mg"
)

type MachineRole string

const (
	MasterRole MachineRole = "Master"
	NodeRole   MachineRole = "Node"
)

func GetMachineRole(machine *clusterapi.Machine) MachineRole {
	if _, found := machine.Labels["set"]; found {
		l := machine.Labels["set"]
		if strings.ToLower(l) == "master" {
			return MasterRole
		}
	}
	return NodeRole
}

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

type MachineProviderConfig struct {
	Name   string   `json:"name,omitempty"`
	Config NodeSpec `json:"config,omitempty"`
}
