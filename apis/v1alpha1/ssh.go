package v1alpha1

type SSHConfig struct {
	PrivateKey      []byte `json:"privateKey,omitempty" protobuf:"bytes,1,opt,name=privateKey"`
	InstanceAddress string `json:"instanceAddress,omitempty" protobuf:"bytes,2,opt,name=instanceAddress"`
	InstancePort    int32  `json:"instancePort,omitempty" protobuf:"varint,3,opt,name=instancePort"`
	User            string `json:"user,omitempty" protobuf:"bytes,4,opt,name=user"`
}
