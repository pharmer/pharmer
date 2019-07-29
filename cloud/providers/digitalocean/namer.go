package digitalocean

import (
	api "pharmer.dev/pharmer/apis/v1alpha1"
)

type namer struct {
	cluster *api.Cluster
}

func (n namer) GenSSHKeyExternalID() string {
	return n.cluster.Name + "-sshkey"
}

func (n namer) LoadBalancerName() string {
	return n.cluster.Name + "-lb"
}
