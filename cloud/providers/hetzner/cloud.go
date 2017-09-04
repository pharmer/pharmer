package hetzner

import (
	"context"
	"strings"
	"time"

	hc "github.com/appscode/go-hetzner"
	"github.com/appscode/go/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/credential"
)

type cloudConnector struct {
	ctx     context.Context
	cluster *api.Cluster
	client  *hc.Client
}

func NewConnector(ctx context.Context, cluster *api.Cluster) (*cloudConnector, error) {
	// TODO: Load once
	cred, err := cloud.Store(ctx).Credentials().Get(cluster.Spec.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.Hetzner{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.New().WithMessagef("Credential %s is invalid. Reason: %v", cluster.Spec.CredentialName, err)
	}

	return &cloudConnector{
		ctx:    ctx,
		client: hc.NewClient(typed.Username(), typed.Password()),
	}, nil
}

func (conn *cloudConnector) waitForInstance(id, status string) (*hc.Transaction, error) {
	attempt := 0
	for {
		cloud.Logger(conn.ctx).Infof("Checking status of instance %v", id)
		tx, _, err := conn.client.Ordering.GetTransaction(id)
		if err != nil {
			return nil, errors.FromErr(err).WithContext(conn.ctx).Err()
		}
		if strings.ToLower(tx.Status) == status {
			return tx, nil
		}
		cloud.Logger(conn.ctx).Infof("Instance %v is %v, waiting...", *tx.ServerIP, tx.Status)
		attempt += 1
		time.Sleep(30 * time.Second)
	}
	//return nil, errors.New().WithMessagef("Failed Hertzner transaction %v", id).Err()
}
