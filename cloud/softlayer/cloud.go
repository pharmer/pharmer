package softlayer

import (
	"fmt"
	"time"

	"github.com/appscode/errors"
	"github.com/appscode/pharmer/contexts"
	"github.com/appscode/pharmer/util/credentialutil"
	"github.com/softlayer/softlayer-go/services"
	"github.com/softlayer/softlayer-go/session"
)

type cloudConnector struct {
	ctx                   *contexts.ClusterContext
	virtualServiceClient  services.Virtual_Guest
	accountServiceClient  services.Account
	securityServiceClient services.Security_Ssh_Key
}

func NewConnector(ctx *contexts.ClusterContext) (*cloudConnector, error) {
	apiKey, ok := ctx.CloudCredential[credentialutil.SoftlayerCredentialApiKey]
	if !ok {
		return nil, errors.New().WithMessagef("Cluster %v credential is missing %v", ctx.Name, credentialutil.SoftlayerCredentialApiKey)
	}
	userName, ok := ctx.CloudCredential[credentialutil.SoftlayerCredentialUsername]
	if !ok {
		return nil, errors.New().WithMessagef("Cluster %v credential is missing %v", ctx.Name, credentialutil.SoftlayerCredentialUsername)
	}

	sess := session.New(userName, apiKey)
	sess.Debug = true
	return &cloudConnector{
		ctx:                   ctx,
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
