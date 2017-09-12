package softlayer

import (
	"context"
	"fmt"
	"time"

	"github.com/appscode/go/errors"
	"github.com/appscode/pharmer/api"
	. "github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/credential"
	"github.com/softlayer/softlayer-go/services"
	"github.com/softlayer/softlayer-go/session"
)

type cloudConnector struct {
	ctx                   context.Context
	cluster               *api.Cluster
	virtualServiceClient  services.Virtual_Guest
	accountServiceClient  services.Account
	securityServiceClient services.Security_Ssh_Key
}

func NewConnector(ctx context.Context, cluster *api.Cluster) (*cloudConnector, error) {
	cred, err := Store(ctx).Credentials().Get(cluster.Spec.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.Softlayer{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.New().WithMessagef("Credential %s is invalid. Reason: %v", cluster.Spec.CredentialName, err)
	}

	sess := session.New(typed.Username(), typed.APIKey())
	sess.Debug = true
	return &cloudConnector{
		ctx:                   ctx,
		cluster:               cluster,
		virtualServiceClient:  services.GetVirtualGuestService(sess),
		accountServiceClient:  services.GetAccountService(sess),
		securityServiceClient: services.GetSecuritySshKeyService(sess),
	}, nil
}

func (conn *cloudConnector) waitForInstance(id int) {
	service := conn.virtualServiceClient.Id(id)

	// Delay to allow transactions to be registered
	for transactions, _ := service.GetActiveTransactions(); len(transactions) > 0; {
		fmt.Print(".")
		time.Sleep(30 * time.Second)
		transactions, _ = service.GetActiveTransactions()
	}
	for yes, _ := service.IsPingable(); !yes; {
		fmt.Print(".")
		time.Sleep(15 * time.Second)
		yes, _ = service.IsPingable()
	}
	for yes, _ := service.IsBackendPingable(); !yes; {
		fmt.Print(".")
		time.Sleep(15 * time.Second)
		yes, _ = service.IsBackendPingable()
	}
}
