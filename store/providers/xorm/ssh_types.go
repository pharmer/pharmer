package xorm

type SSHKey struct {
	ID          int64
	Name        string `xorm:"not null 'name'"`
	ClusterID   int64  `xorm:"NOT NULL 'cluster_id'"`
	ClusterName string `xorm:"not null 'cluster_name'"`
	UID         string `xorm:"not null 'uid'"`
	PublicKey   string `xorm:"text not null 'public_key'"`
	PrivateKey  string `xorm:"text not null 'private_key'"`

	CreatedUnix int64  `xorm:"INDEX created"`
	UpdatedUnix int64  `xorm:"INDEX updated"`
	DeletedUnix *int64 `xorm:"null"`
}

func (SSHKey) TableName() string {
	return "ac_cluster_ssh"
}

func encodeSSHKey(pub, priv []byte) *SSHKey {
	return &SSHKey{
		PublicKey:   string(pub),
		PrivateKey:  string(priv),
		DeletedUnix: nil,
	}
}

func decodeSSHKey(in *SSHKey) ([]byte, []byte, error) {
	return []byte(in.PublicKey), []byte(in.PrivateKey), nil
}
