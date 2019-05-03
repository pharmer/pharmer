package cloud

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/2017-03-09/resources/mgmt/subscriptions"
	aauthz "github.com/Azure/azure-sdk-for-go/profiles/latest/authorization/mgmt/authorization"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/graphrbac/graphrbac"
	"github.com/Azure/azure-sdk-for-go/profiles/preview/preview/apimanagement/mgmt/apimanagement"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	azdate "github.com/Azure/go-autorest/autorest/date"
	"github.com/appscode/go/term"
	"github.com/appscode/go/types"
	"github.com/pborman/uuid"
	api "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	"github.com/pharmer/cloud/pkg/credential"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	azureNativeApplicationID = "a6fa51f3-f8b6-4eb5-833a-58a706552eae"
	azureTenantID            = "772268e5-d940-4bf6-be82-1c4a09a67f5d"
	RetryInterval            = 5 * time.Second
	RetryTimeout             = 15 * time.Minute
)

func getSptFromDeviceFlow(oauthConfig adal.OAuthConfig, clientID, resource string) (*adal.ServicePrincipalToken, error) {
	oauthClient := &autorest.Client{}
	deviceCode, err := adal.InitiateDeviceAuth(oauthClient, oauthConfig, clientID, resource)
	if err != nil {
		return nil, errors.Errorf("Failed to start device auth flow: %s", err)
	}
	fmt.Println(*deviceCode.Message)

	token, err := adal.WaitForUserCompletion(oauthClient, deviceCode)
	if err != nil {
		return nil, errors.Errorf("Failed to finish device auth flow: %s", err)
	}

	spt, err := adal.NewServicePrincipalTokenFromManualToken(
		oauthConfig,
		clientID,
		resource,
		*token)
	if err != nil {
		return nil, errors.Errorf("Failed to get oauth token from device flow: %v", err)
	}
	return spt, nil
}

func IssueAzureCredential(name string) (*api.Credential, error) {
	oauthConfig, err := adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, azureTenantID)
	if err != nil {
		return nil, err
	}

	spt, err := getSptFromDeviceFlow(*oauthConfig, azureNativeApplicationID, azure.PublicCloud.ServiceManagementEndpoint)
	if err != nil {
		return nil, err
	}

	client := autorest.NewClientWithUserAgent(subscriptions.UserAgent())
	client.Authorizer = autorest.NewBearerAuthorizer(spt)

	tenantsClient := subscriptions.NewTenantsClientWithBaseURI(subscriptions.DefaultBaseURI)
	tenantsClient.Authorizer = autorest.NewBearerAuthorizer(spt)
	tenants, err := tenantsClient.List(context.TODO())
	if err != nil {
		return nil, err
	}
	tenantID := types.String((tenants.Values())[0].TenantID)

	userOauthConfig, err := adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, tenantID)
	if err != nil {
		return nil, err
	}

	userSpt, err := adal.NewServicePrincipalTokenFromManualToken(
		*userOauthConfig,
		azureNativeApplicationID,
		azure.PublicCloud.ServiceManagementEndpoint,
		spt.Token())
	if err != nil {
		return nil, err
	}

	err = userSpt.RefreshExchange(azure.PublicCloud.ServiceManagementEndpoint)
	if err != nil {
		return nil, err
	}

	userClient := autorest.NewClientWithUserAgent(subscriptions.UserAgent())
	userClient.Authorizer = autorest.NewBearerAuthorizer(userSpt)

	// TODO(tamal): Fix it!
	userSubsClient := apimanagement.NewGroupClientWithBaseURI(subscriptions.DefaultBaseURI, "")
	userSubsClient.Authorizer = autorest.NewBearerAuthorizer(userSpt)

	// TODO(tamal): Fix it!
	subs, err := userSubsClient.ListByServiceComplete(context.TODO(), "", "", "", nil, nil)
	if err != nil {
		return nil, err
	}
	subscriptionID := types.String(subs.Value().ID)

	graphSpt, err := adal.NewServicePrincipalTokenFromManualToken(
		*userOauthConfig,
		azureNativeApplicationID,
		azure.PublicCloud.GraphEndpoint,
		userSpt.Token())
	if err != nil {
		return nil, err
	}

	err = graphSpt.RefreshExchange(azure.PublicCloud.GraphEndpoint)
	if err != nil {
		return nil, err
	}

	graphClient := autorest.NewClientWithUserAgent(graphrbac.UserAgent())
	graphClient.Authorizer = autorest.NewBearerAuthorizer(graphSpt)

	clientSecret := uuid.NewRandom().String()
	start := azdate.Time{Time: time.Now()}
	end := azdate.Time{Time: time.Now().Add(365 * 24 * time.Hour)}
	cred := graphrbac.PasswordCredential{
		StartDate: &start,
		EndDate:   &end,
		Value:     types.StringP(clientSecret),
	}

	appClient := graphrbac.NewApplicationsClientWithBaseURI(graphrbac.DefaultBaseURI, tenantID)
	appClient.Authorizer = autorest.NewBearerAuthorizer(graphSpt)

	app, err := appClient.Create(context.TODO(), graphrbac.ApplicationCreateParameters{
		AvailableToOtherTenants: types.FalseP(),
		DisplayName:             types.StringP(name),
		Homepage:                types.StringP("http://" + name),
		IdentifierUris:          &[]string{"http://" + name},
		PasswordCredentials:     &[]graphrbac.PasswordCredential{cred},
	})
	if err != nil {
		return nil, err
	}
	clientID := *app.AppID

	spClient := graphrbac.NewServicePrincipalsClientWithBaseURI(graphrbac.DefaultBaseURI, tenantID)
	spClient.Authorizer = autorest.NewBearerAuthorizer(graphSpt)
	sp, err := spClient.Create(context.TODO(), graphrbac.ServicePrincipalCreateParameters{
		AppID:          types.StringP(clientID),
		AccountEnabled: types.TrueP(),
	})
	if err != nil {
		return nil, err
	}

	roleDefClient := aauthz.NewRoleDefinitionsClientWithBaseURI(aauthz.DefaultBaseURI, subscriptionID)
	roleDefClient.Authorizer = autorest.NewBearerAuthorizer(userSpt)

	roles, err := roleDefClient.List(context.TODO(), "subscriptions/"+subscriptionID, `roleName eq 'Contributor'`)
	if err != nil {
		return nil, err
	}
	if len(roles.Values()) == 0 {
		term.Fatalln("Can't find Contributor role.")
	}
	roleID := (roles.Values())[0].ID

	roleAssignClient := aauthz.NewRoleAssignmentsClientWithBaseURI(aauthz.DefaultBaseURI, subscriptionID)
	roleAssignClient.Authorizer = autorest.NewBearerAuthorizer(userSpt)

	wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		roleAssignmentName := uuid.NewRandom().String()
		_, err := roleAssignClient.Create(context.TODO(), "subscriptions/"+subscriptionID, roleAssignmentName, aauthz.RoleAssignmentCreateParameters{
			Properties: &aauthz.RoleAssignmentProperties{
				PrincipalID:      sp.ObjectID,
				RoleDefinitionID: roleID,
			},
		})
		if err != nil {
			return false, nil
		}
		return true, nil
	})

	return &api.Credential{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: api.CredentialSpec{
			Provider: "Azure",
			Data: map[string]string{
				credential.AzureSubscriptionID: subscriptionID,
				credential.AzureTenantID:       tenantID,
				credential.AzureClientID:       clientID,
				credential.AzureClientSecret:   clientSecret,
			},
		},
	}, nil
}
