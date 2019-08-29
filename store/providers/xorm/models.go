package xorm

import (
	"gomodules.xyz/secrets/xkms"
	"k8s.io/klog/klogr"
)

var (
	tables []interface{}
	log    = klogr.New().WithName("[xorm-store]")
)

func init() {
	tables = append(tables,
		new(Certificate),
		new(Credential),
		new(Cluster),
		new(Machine),
		new(SSHKey),
		new(xkms.SecretKey),
	)
}
