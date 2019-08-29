package xorm

import (
	"fmt"

	"gomodules.xyz/secrets/types"
)

type SSHKey struct {
	ID          int64
	Name        string `xorm:"not null 'name'"`
	ClusterID   int64  `xorm:"NOT NULL 'cluster_id'"`
	ClusterName string `xorm:"not null 'cluster_name'"`
	UID         string `xorm:"not null 'uid'"`
	PublicKey   string `xorm:"text not null 'public_key'"`
	PrivateKey  []byte `xorm:"blob not null 'private_key'"`
	SecretID    string

	CreatedUnix int64  `xorm:"INDEX created"`
	UpdatedUnix int64  `xorm:"INDEX updated"`
	DeletedUnix *int64 `xorm:"null"`
}

func (SSHKey) TableName() string {
	return "ac_cluster_ssh"
}

func EncodeSSHKey(pub, priv []byte) (*SSHKey, error) {
	secretId := types.RotateQuarterly()
	cipher, err := encryptData(secretId, priv)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt: %v", err)
	}
	return &SSHKey{
		PublicKey:   string(pub),
		PrivateKey:  cipher,
		SecretID:    secretId,
		DeletedUnix: nil,
	}, nil
}

func DecodeSSHKey(in *SSHKey) ([]byte, []byte, error) {
	data, err := decryptData(in.SecretID, in.PrivateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decrypt: %v", err)
	}
	return []byte(in.PublicKey), data, nil
}
