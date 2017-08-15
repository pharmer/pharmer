package softlayer

import (
	"fmt"
	"time"

	"github.com/appscode/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/credential"
	"github.com/softlayer/softlayer-go/services"
	"github.com/softlayer/softlayer-go/session"
)

type cloudConnector struct {
	ctx                   *api.Cluster
	virtualServiceClient  services.Virtual_Guest
	accountServiceClient  services.Account
	securityServiceClient services.Security_Ssh_Key
}

func NewConnector(ctx *api.Cluster) (*cloudConnector, error) {
	apiKey, ok := ctx.CloudCredential[credential.SoftlayerAPIKey]
	if !ok {
		return nil, errors.New().WithMessagef("Cluster %v credential is missing %v", ctx.Name, credential.SoftlayerAPIKey)
	}
	userName, ok := ctx.CloudCredential[credential.SoftlayerUsername]
	if !ok {
		return nil, errors.New().WithMessagef("Cluster %v credential is missing %v", ctx.Name, credential.SoftlayerUsername)
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
