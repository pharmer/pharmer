package xorm

import (
	"context"

	"gocloud.dev/secrets"
	"gomodules.xyz/secrets/xkms"
)

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
