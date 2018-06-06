package cloud

const (
	EtcdCACertAndKeyBaseName = "etcd/ca"
	EtcdCACertName           = "etcd/ca.crt"
	EtcdCAKeyName            = "etcd/ca.key"

	EtcdServerCertAndKeyBaseName = "etcd/server"
	EtcdServerCertName           = "etcd-server.crt"
	EtcdServerKeyName            = "etcd-server.key"

	EtcdImage = "pharmer/lector:0.1.0-alpha.7"
)

type EtcdCert struct {
}
