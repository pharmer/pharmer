package xorm

import (
	"encoding/json"

	"gomodules.xyz/secrets/types"
	cloudapi "pharmer.dev/cloud/pkg/apis/cloud/v1"
)

type Credential struct {
	ID       int64  `xorm:"pk autoincr"`
	OwnerID  int64  `xorm:"UNIQUE(s)"`
	Name     string `xorm:"UNIQUE(s) INDEX NOT NULL"`
	UID      string `xorm:"uid UNIQUE"`
	Data     []byte `xorm:"blob NOT NULL"`
	SecretID string

	CreatedUnix int64  `xorm:"INDEX created"`
	UpdatedUnix int64  `xorm:"INDEX updated"`
	DeletedUnix *int64 `xorm:"null"`
}

func (Credential) TableName() string {
	return "ac_cluster_credential"
}

func EncodeCredential(in *cloudapi.Credential) (*Credential, error) {
	secretId := types.RotateQuarterly()
	data, err := json.Marshal(in)
	if err != nil {
		log.Error(err, "failed to marshal credential")
		return nil, err
	}

	cipher, err := encryptData(secretId, data)
	if err != nil {
		log.Error(err, "failed to encrypt credential")
		return nil, err
	}

	return &Credential{
		Name:        in.Name,
		Data:        cipher,
		SecretID:    secretId,
		DeletedUnix: nil,
	}, nil
}

func DecodeCredential(in *Credential) (*cloudapi.Credential, error) {
	data, err := decryptData(in.SecretID, in.Data)
	if err != nil {
		log.Error(err, "failed to decrypt credential")
		return nil, err
	}
	var obj cloudapi.Credential
	if err := json.Unmarshal(data, &obj); err != nil {
		log.Error(err, "failed to unmarshal credential")
		return nil, err
	}
	return &obj, nil
}
