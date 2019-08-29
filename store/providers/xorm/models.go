package xorm

import (
	"context"

	"gocloud.dev/secrets"
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

func encryptData(secretID string, data []byte) ([]byte, error) {
	ctx := context.Background()
	urlstr := xkms.Scheme + "://" + secretID
	keeper, err := secrets.OpenKeeper(ctx, urlstr)
	if err != nil {
		return nil, err
	}
	defer keeper.Close()

	return keeper.Encrypt(ctx, data)
}

func decryptData(secretID string, cipher []byte) ([]byte, error) {
	ctx := context.Background()
	urlstr := xkms.Scheme + "://" + secretID
	keeper, err := secrets.OpenKeeper(ctx, urlstr)
	if err != nil {
		return nil, err
	}
	defer keeper.Close()

	return keeper.Decrypt(ctx, cipher)
}
