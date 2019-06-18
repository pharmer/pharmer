package aks

import (
	"context"
	"fmt"

	ms "github.com/Azure/azure-sdk-for-go/profiles/latest/containerservice/mgmt/containerservice"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	cs "github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2019-04-30/containerservice"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-03-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/appscode/go/log"
	"github.com/appscode/go/types"
	"github.com/appscode/go/wait"
	"github.com/pharmer/cloud/pkg/credential"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
)

type cloudConnector struct {
	*cloud.Scope

	namer namer

	availabilitySetsClient compute.AvailabilitySetsClient
	groupsClient           resources.GroupsClient
	managedClient          ms.ManagedClustersClient
}

func newConnector(cm *ClusterManager) (*cloudConnector, error) {
	cluster := cm.Cluster
	cred, err := cm.StoreProvider.Credentials().Get(cluster.Spec.Config.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.Azure{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.Wrapf(err, "credential %s is invalid", cluster.Spec.Config.CredentialName)
	}

	baseURI := azure.PublicCloud.ResourceManagerEndpoint
	config, err := adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, typed.TenantID())
	if err != nil {
		return nil, err
	}

	spt, err := adal.NewServicePrincipalToken(*config, typed.ClientID(), typed.ClientSecret(), baseURI)
	if err != nil {
		return nil, err
	}

	client := autorest.NewClientWithUserAgent(fmt.Sprintf("Azure-SDK-for-Go/%s", compute.Version()))
	client.Authorizer = autorest.NewBearerAuthorizer(spt)

	availabilitySetsClient := compute.NewAvailabilitySetsClientWithBaseURI(baseURI, typed.SubscriptionID())
	availabilitySetsClient.Authorizer = autorest.NewBearerAuthorizer(spt)

	groupsClient := resources.NewGroupsClientWithBaseURI(baseURI, typed.SubscriptionID())
	groupsClient.Authorizer = autorest.NewBearerAuthorizer(spt)

	managedClient := ms.NewManagedClustersClient(typed.SubscriptionID())
	managedClient.Authorizer = autorest.NewBearerAuthorizer(spt)

	return &cloudConnector{
		Scope:                  cm.Scope,
		namer:                  cm.namer,
		availabilitySetsClient: availabilitySetsClient,
		groupsClient:           groupsClient,
		managedClient:          managedClient,
	}, nil
}

func (conn *cloudConnector) getResourceGroup() (bool, error) {
	_, err := conn.groupsClient.Get(context.TODO(), conn.namer.ResourceGroupName())
	return err == nil, err
}

func (conn *cloudConnector) ensureResourceGroup() (resources.Group, error) {
	req := resources.Group{
		Name:     types.StringP(conn.namer.ResourceGroupName()),
		Location: types.StringP(conn.Cluster.Spec.Config.Cloud.Zone),
		Tags: map[string]*string{
			"KubernetesCluster": types.StringP(conn.Cluster.Name),
		},
	}
	return conn.groupsClient.CreateOrUpdate(context.TODO(), conn.namer.ResourceGroupName(), req)
}

func (conn *cloudConnector) deleteResourceGroup() error {
	_, err := conn.groupsClient.Delete(context.TODO(), conn.namer.ResourceGroupName())
	log.Infof("Resource group %v deleted", conn.namer.ResourceGroupName())
	return err
}

func (conn *cloudConnector) upsertAKS(agentPools []cs.ManagedClusterAgentPoolProfile) error {
	cred, err := conn.StoreProvider.Credentials().Get(conn.Cluster.Spec.Config.CredentialName)
	if err != nil {
		return err
	}
	typed := credential.Azure{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return errors.Wrapf(err, "credential %s is invalid", conn.Cluster.Spec.Config.CredentialName)
	}

	container := cs.ManagedCluster{
		Name:     &conn.Cluster.Name,
		Location: types.StringP(conn.Cluster.Spec.Config.Cloud.Zone),
		ManagedClusterProperties: &cs.ManagedClusterProperties{
			DNSPrefix: types.StringP(conn.Cluster.Name),
			//Fqdn:              types.StringP(conn.Cluster.Name),
			KubernetesVersion: types.StringP(conn.Cluster.Spec.Config.KubernetesVersion),
			ServicePrincipalProfile: &cs.ManagedClusterServicePrincipalProfile{
				ClientID: types.StringP(typed.ClientID()),
				Secret:   types.StringP(typed.ClientSecret()),
			},

			AgentPoolProfiles: &agentPools,
			LinuxProfile: &cs.LinuxProfile{
				AdminUsername: types.StringP(conn.namer.AdminUsername()),
				SSH: &cs.SSHConfiguration{
					PublicKeys: &[]cs.SSHPublicKey{
						{
							KeyData: types.StringP(string(conn.Certs.SSHKey.PublicKey)),
						},
					},
				},
			},
		},
	}

	_, err = conn.managedClient.CreateOrUpdate(context.Background(), conn.namer.ResourceGroupName(), conn.Cluster.Name, container)
	if err != nil {
		return err
	}

	return conn.WaitForClusterOperation()
}

func (conn *cloudConnector) WaitForClusterOperation() error {
	attempt := 0
	return wait.PollImmediate(api.RetryInterval, api.RetryTimeout, func() (bool, error) {
		attempt++
		r, err := conn.managedClient.Get(context.Background(), conn.namer.ResourceGroupName(), conn.Cluster.Name)
		if err != nil {
			return false, nil
		}
		log.Infof("Attempt %v: Operation %v is %v ...", attempt, *r.Name, *r.ProvisioningState)
		if *r.ProvisioningState == "Succeeded" {
			return true, nil
		}
		return false, nil
	})
}

func (conn *cloudConnector) deleteAKS() error {
	_, err := conn.managedClient.Delete(context.Background(), conn.namer.ResourceGroupName(), conn.Cluster.Name)
	return err
}
