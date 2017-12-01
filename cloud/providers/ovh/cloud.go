package ovh

import (
	"context"

	"github.com/appscode/go/errors"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/images"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/pagination"
	api "github.com/pharmer/pharmer/apis/v1alpha1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/credential"
)

const auth_url = "https://auth.cloud.ovh.net/"

type cloudConnector struct {
	ctx     context.Context
	cluster *api.Cluster
	client  *gophercloud.ServiceClient
	namer   namer
}

var _ InstanceManager = &cloudConnector{}

func NewConnector(ctx context.Context, cluster *api.Cluster) (*cloudConnector, error) {
	cred, err := Store(ctx).Credentials().Get(cluster.Spec.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.Ovh{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.New().WithMessagef("Credential %s is invalid. Reason: %v", cluster.Spec.CredentialName, err)
	}

	opts := gophercloud.AuthOptions{
		IdentityEndpoint: auth_url + "v2.0",
		Username:         typed.Username(),
		Password:         typed.Password(),
		TenantID:         typed.TenantID(),
	}

	provider, err := openstack.AuthenticatedClient(opts)
	if err != nil {
		return nil, errors.New().WithMessagef("Credential %s is not authenticated. Reason: %v", cluster.Spec.CredentialName, err)
	}

	client, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
		Name:   "nova",
		Region: cluster.Spec.Cloud.Region,
	})
	if err != nil {
		return nil, err
	}
	return &cloudConnector{
		ctx:     ctx,
		cluster: cluster,
		namer:   namer{cluster},
		client:  client,
	}, nil
}

func (conn *cloudConnector) detectInstanceImage() error {
	opts := images.ListOpts{ChangesSince: "2014-01-01T01:02:03Z", Name: "Ubuntu 16.04"}
	pager := images.ListDetail(conn.client, opts)
	return pager.EachPage(func(page pagination.Page) (bool, error) {
		imageList, err := images.ExtractImages(page)
		if err != nil {
			return false, err
		}
		for _, i := range imageList {
			if i.Name == "Ubuntu 16.04" {
				conn.cluster.Spec.Cloud.InstanceImage = i.ID
				return true, nil

			}
		}
		return false, nil
	})
}

func (conn *cloudConnector) CreateInstance(name, token string, ng *api.NodeGroup) (*api.NodeInfo, error) {
	/*opts := servers.CreateOpts{

	}*/
	return nil, nil
}

func (conn *cloudConnector) DeleteInstanceByProviderID(providerID string) error {
	return nil
}


func (conn *cloudConnector) getPublicKey() (bool, string, error) {
	/*pager := keypairs.List(conn.client)
	pager.EachPage(func(page pagination.Page) (bool, error) {

	})
	keys, err := conn.client.L()
	if err != nil {
		return false, "", err
	}
	for _, key := range keys {
		if key.Name == conn.cluster.Spec.Cloud.SSHKeyName {
			return true, key.ID, nil
		}
	}*/
	return false, "", nil
}

func (conn *cloudConnector)deleteSSHKey(sshKeyName string) error  {
	return keypairs.Delete(conn.client, sshKeyName).ExtractErr()
}

func (conn *cloudConnector) importPublicKey() error {
	Logger(conn.ctx).Infof("Adding SSH public key")
	opts := keypairs.CreateOpts{
		Name: conn.cluster.Spec.Cloud.SSHKeyName,
		PublicKey: string(SSHKey(conn.ctx).PublicKey),
	}
	resp, err := keypairs.Create(conn.client, opts).Extract()
	if err != nil {
		return errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	conn.cluster.Status.Cloud.SShKeyExternalID = resp.Name
	Logger(conn.ctx).Infof("New ssh key with name %v and id %v created", conn.cluster.Spec.Cloud.SSHKeyName, resp.Name)
	return nil
}
