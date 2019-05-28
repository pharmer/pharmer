package cloud

type namer struct {
	clusterName string // *api.Cluster
}

func (n namer) GenSSHKeyExternalID() string {
	return n.clusterName + "-sshkey"
}
