package cloud

const (
	EtcdCACertAndKeyBaseName = "etcd/ca"
	EtcdCACertName           = "etcd/ca.crt"
	EtcdCAKeyName            = "etcd/ca.key"

	EtcdServerCertAndKeyBaseName = "etcd/server"
	EtcdServerCertName           = "etcd-server.crt"
	EtcdServerKeyName            = "etcd-server.key"

	EtcdImage = "pharmer/lector:1.10.1-alpha.4"
)

type EtcdCert struct {
}
