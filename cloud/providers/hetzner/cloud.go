package hetzner

import (
	"strings"
	"time"

	"github.com/appscode/errors"
	hc "github.com/appscode/go-hetzner"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/context"
	"github.com/appscode/pharmer/credential"
)

type cloudConnector struct {
	ctx     context.Context
	cluster *api.Cluster
	client  *hc.Client
}

func NewConnector(ctx context.Context, cluster *api.Cluster) (*cloudConnector, error) {
	username, ok := cluster.CloudCredential[credential.HertznerUsername]
	if !ok {
		return nil, errors.New().WithMessagef("Cluster %v credential is missing %v", cluster.Name, credential.HertznerUsername)
	}
	password, ok := cluster.CloudCredential[credential.HertznerPassword]
	if !ok {
		return nil, errors.New().WithMessagef("Cluster %v credential is missing %v", cluster.Name, credential.HertznerPassword)
	}
	return &cloudConnector{
		ctx:    ctx,
		client: hc.NewClient(username, password),
	}, nil
}

func (conn *cloudConnector) waitForInstance(id, status string) (*hc.Transaction, error) {
	attempt := 0
	for {
		conn.ctx.Logger().Infof("Checking status of instance %v", id)
		tx, _, err := conn.client.Ordering.GetTransaction(id)
		if err != nil {
			return nil, errors.FromErr(err).WithContext(conn.ctx).Err()
		}
		if strings.ToLower(tx.Status) == status {
			return tx, nil
		}
		conn.ctx.Logger().Infof("Instance %v is %v, waiting...", *tx.ServerIP, tx.Status)
		attempt += 1
		time.Sleep(30 * time.Second)
	}
	return nil, errors.New().WithMessagef("Failed Hertzner transaction %v", id).Err()
}
