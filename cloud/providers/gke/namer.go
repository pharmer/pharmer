package gke

import (
	api "pharmer.dev/pharmer/apis/v1alpha1"
)

type namer struct {
	cluster *api.Cluster
}

func (n namer) AdminUsername() string {
	return "pharmer"
}

func (n namer) GenSSHKeyExternalID() string {
	return n.cluster.Name + "-sshkey"
}
