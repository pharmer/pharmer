package hetzner

import (
	"strings"
	"time"

	"github.com/appscode/errors"
	hc "github.com/appscode/go-hetzner"
	"github.com/appscode/pharmer/contexts"
	"github.com/appscode/pharmer/util/credentialutil"
)

type cloudConnector struct {
	ctx    *contexts.ClusterContext
	client *hc.Client
}

func NewConnector(ctx *contexts.ClusterContext) (*cloudConnector, error) {
	username, ok := ctx.CloudCredential[credentialutil.HertznerCredentialUsername]
	if !ok {
		return nil, errors.New().WithMessagef("Cluster %v credential is missing %v", ctx.Name, credentialutil.HertznerCredentialUsername)
	}
	password, ok := ctx.CloudCredential[credentialutil.HertznerCredentialPassword]
	if !ok {
		return nil, errors.New().WithMessagef("Cluster %v credential is missing %v", ctx.Name, credentialutil.HertznerCredentialPassword)
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
