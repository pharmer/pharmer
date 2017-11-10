package framework

import (

	"github.com/appscode/go/crypto/ssh"
)

func (c *sshInvocation) GetName() string  {
	return "storage-test"
}

func (c *sshInvocation) GetSkeleton() (*ssh.SSHKey, error) {
	return ssh.NewSSHKeyPair()
}

func (c *sshInvocation) Create(key *ssh.SSHKey) error  {
	return c.Storage.SSHKeys(c.clusterName).Create(c.GetName(), key.PublicKey, key.PrivateKey)
}
