package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	MachineSlecetor = "cloud.appscode.com/mg"
)

type NodeSpec struct {
	SKU              string            `json:"sku,omitempty" protobuf:"bytes,1,opt,name=sku"`
	DiskType         string            `json:"nodeDiskType,omitempty" protobuf:"bytes,2,opt,name=nodeDiskType"`
	DiskSize         int64             `json:"nodeDiskSize,omitempty" protobuf:"varint,3,opt,name=nodeDiskSize"`
	ExternalIPType   IPType            `json:"externalIPType,omitempty" protobuf:"bytes,4,opt,name=externalIPType,casttype=IPType"`
	KubeletExtraArgs map[string]string `json:"kubeletExtraArgs,omitempty" protobuf:"bytes,5,rep,name=kubeletExtraArgs"`
	Type             NodeType          `json:"type,omitempty" protobuf:"varint,6,opt,name=type"`
	SpotPriceMax     float64           `json:"spotPriceMax,omitempty" protobuf:"fixed64,7,opt,name=spotPriceMax"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type NodeInfo struct {
	metav1.TypeMeta `json:",inline,omitempty"`
	Name            string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	ExternalID      string `json:"externalID,omitempty" protobuf:"bytes,2,opt,name=externalID"`
	PublicIP        string `json:"publicIP,omitempty" protobuf:"bytes,3,opt,name=publicIP"`
	PrivateIP       string `json:"privateIP,omitempty" protobuf:"bytes,4,opt,name=privateIP"`
	DiskId          string `json:"diskID,omitempty" protobuf:"bytes,5,opt,name=diskID"`
}

type MachineProviderConfig struct {
	Name   string   `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	Config NodeSpec `json:"config,omitempty" protobuf:"bytes,2,opt,name=config"`
}
