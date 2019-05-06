package vultr

import (
	"github.com/appscode/go/crypto/rand"
	api "github.com/pharmer/pharmer/apis/v1beta1"
)

type namer struct {
	cluster *api.Cluster
}

func (n namer) MasterName() string {
	return n.cluster.Name + "-master"
}

// Deprecated
func (n namer) GenNodeName(ng string) string {
	return rand.WithUniqSuffix(ng)
}

func (n namer) GenSSHKeyExternalID() string {
	return n.cluster.Name + "-sshkey"
}

func (n namer) ReserveIPName() string {
	return n.cluster.Name + "-master-ip"
}

func (n namer) StartupScriptName(ng, role string) string {
	return n.cluster.Name + "-" + ng + "-" + role
}
