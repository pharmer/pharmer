package v1beta1

type SSHConfig struct {
	PrivateKey []byte `json:"privateKey,omitempty"`
	HostIP     string `json:"hostIP,omitempty"`
	HostPort   int32  `json:"hostPort,omitempty"`
	User       string `json:"user,omitempty"`
}
