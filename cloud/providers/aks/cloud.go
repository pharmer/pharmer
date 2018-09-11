package aks

import (
	"context"
	"fmt"
	"regexp"

	ms "github.com/Azure/azure-sdk-for-go/profiles/latest/containerservice/mgmt/containerservice"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2017-12-01/compute"
	cs "github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2017-09-30/containerservice"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2017-05-10/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	. "github.com/appscode/go/context"
	. "github.com/appscode/go/types"
	"github.com/appscode/go/wait"
	api "github.com/pharmer/pharmer/apis/v1alpha1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/credential"
	"github.com/pkg/errors"
)

const (
	machineIDTemplate = "/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s"
	CloudProviderName = "azure"
)

var providerIDRE = regexp.MustCompile(`^` + CloudProviderName + `://(?:.*)/Microsoft.Compute/virtualMachines/(.+)$`)

type cloudConnector struct {
	ctx     context.Context
	cluster *api.Cluster
	namer   namer

	availabilitySetsClient compute.AvailabilitySetsClient
	groupsClient           resources.GroupsClient
	managedClient          ms.ManagedClustersClient
}

func NewConnector(ctx context.Context, cluster *api.Cluster) (*cloudConnector, error) {
	cred, err := Store(ctx).Credentials().Get(cluster.Spec.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.Azure{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.Wrapf(err, "credential %s is invalid", cluster.Spec.CredentialName)
	}

	baseURI := azure.PublicCloud.ResourceManagerEndpoint
	config, err := adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, typed.TenantID())
	if err != nil {
		return nil, errors.Wrap(err, ID(ctx))
	}

	spt, err := adal.NewServicePrincipalToken(*config, typed.ClientID(), typed.ClientSecret(), baseURI)
	if err != nil {
		return nil, errors.Wrap(err, ID(ctx))
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
		cluster:                cluster,
		ctx:                    ctx,
		availabilitySetsClient: availabilitySetsClient,
		groupsClient:           groupsClient,
		managedClient:          managedClient,
	}, nil
}

func (conn *cloudConnector) detectUbuntuImage() error {
	conn.cluster.Spec.Cloud.OS = string(cs.Linux)
	return nil
}

func (conn *cloudConnector) getResourceGroup() (bool, error) {
	_, err := conn.groupsClient.Get(context.TODO(), conn.namer.ResourceGroupName())
	return err == nil, err
}

func (conn *cloudConnector) ensureResourceGroup() (resources.Group, error) {
	req := resources.Group{
		Name:     StringP(conn.namer.ResourceGroupName()),
		Location: StringP(conn.cluster.Spec.Cloud.Zone),
		Tags: map[string]*string{
			"KubernetesCluster": StringP(conn.cluster.Name),
		},
	}
	return conn.groupsClient.CreateOrUpdate(context.TODO(), conn.namer.ResourceGroupName(), req)
}

func (conn *cloudConnector) getAvailabilitySet() (compute.AvailabilitySet, error) {
	return conn.availabilitySetsClient.Get(context.TODO(), conn.namer.ResourceGroupName(), conn.namer.AvailabilitySetName())
}

func (conn *cloudConnector) ensureAvailabilitySet() (compute.AvailabilitySet, error) {
	name := conn.namer.AvailabilitySetName()
	req := compute.AvailabilitySet{
		Name:     StringP(name),
		Location: StringP(conn.cluster.Spec.Cloud.Zone),
		Tags: map[string]*string{
			"KubernetesCluster": StringP(conn.cluster.Name),
		},
	}
	return conn.availabilitySetsClient.CreateOrUpdate(context.TODO(), conn.namer.ResourceGroupName(), name, req)
}

func (conn *cloudConnector) deleteResourceGroup() error {
	_, err := conn.groupsClient.Delete(context.TODO(), conn.namer.ResourceGroupName())
	Logger(conn.ctx).Infof("Resource group %v deleted", conn.namer.ResourceGroupName())
	return err
}

func (conn *cloudConnector) upsertAKS(agentPools []cs.AgentPoolProfile) error {
	cred, err := Store(conn.ctx).Credentials().Get(conn.cluster.Spec.CredentialName)
	if err != nil {
		return err
	}
	typed := credential.Azure{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return errors.Wrapf(err, "credential %s is invalid", conn.cluster.Spec.CredentialName)
	}

	container := cs.ManagedCluster{
		Name:     &conn.cluster.Name,
		Location: StringP(conn.cluster.Spec.Cloud.Zone),
		ManagedClusterProperties: &cs.ManagedClusterProperties{
			DNSPrefix: StringP(conn.cluster.Name),
			//Fqdn:              StringP(conn.cluster.Name),
			KubernetesVersion: StringP(conn.cluster.Spec.KubernetesVersion),
			ServicePrincipalProfile: &cs.ServicePrincipalProfile{
				ClientID: StringP(typed.ClientID()),
				Secret:   StringP(typed.ClientSecret()),
			},

			AgentPoolProfiles: &agentPools,
			LinuxProfile: &cs.LinuxProfile{
				AdminUsername: StringP(conn.namer.AdminUsername()),
				SSH: &cs.SSHConfiguration{
					PublicKeys: &[]cs.SSHPublicKey{
						{
							KeyData: StringP(string(SSHKey(conn.ctx).PublicKey)),
						},
					},
				},
			},
		},
	}

	_, err = conn.managedClient.CreateOrUpdate(context.Background(), conn.namer.ResourceGroupName(), conn.cluster.Name, container)
	if err != nil {
		return err
	}

	return conn.WaitForClusterOperation()
}

func (conn *cloudConnector) WaitForClusterOperation() error {
	attempt := 0
	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		attempt++
		r, err := conn.managedClient.Get(context.Background(), conn.namer.ResourceGroupName(), conn.cluster.Name)
		if err != nil {
			return false, nil
		}
		Logger(conn.ctx).Infof("Attempt %v: Operation %v is %v ...", attempt, *r.Name, *r.ProvisioningState)
		if *r.ProvisioningState == "Succeeded" {
			return true, nil
		}
		return false, nil
	})
}

func (conn *cloudConnector) deleteAKS() error {
	_, err := conn.managedClient.Delete(context.Background(), conn.namer.ResourceGroupName(), conn.cluster.Name)
	return err
}

func (conn *cloudConnector) getUpgradeProfile() (bool, error) {
	resp, err := conn.managedClient.GetUpgradeProfile(context.Background(), conn.namer.ResourceGroupName(), conn.cluster.Name)
	if err != nil {
		return false, err
	}
	if *resp.ControlPlaneProfile.KubernetesVersion == conn.cluster.Spec.KubernetesVersion {
		return false, nil
	}
	return true, nil
}

func (conn *cloudConnector) upgradeCluster() error {
	cluster, err := conn.managedClient.Get(context.Background(), conn.namer.ResourceGroupName(), conn.cluster.Name)
	if err != nil {
		return err
	}
	cluster.KubernetesVersion = StringP(conn.cluster.Spec.KubernetesVersion)
	_, err = conn.managedClient.CreateOrUpdate(context.Background(), conn.namer.ResourceGroupName(), conn.cluster.Name, cluster)
	if err != nil {
		return err
	}
	return conn.WaitForClusterOperation()
}
