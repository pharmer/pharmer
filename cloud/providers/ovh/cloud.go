package ovh

import (
	"context"

	"github.com/appscode/go/errors"
	. "github.com/appscode/go/types"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/secgroups"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/images"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/pagination"
	api "github.com/pharmer/pharmer/apis/v1alpha1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/credential"
)

const auth_url = "https://auth.cloud.ovh.net/"

type cloudConnector struct {
	ctx           context.Context
	cluster       *api.Cluster
	computeClient *gophercloud.ServiceClient
	networkClient *gophercloud.ServiceClient
	namer         namer
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

	computeClient, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
		Name:   "nova",
		Region: cluster.Spec.Cloud.Region,
	})
	if err != nil {
		return nil, err
	}

	networkClient, err := openstack.NewNetworkV2(provider, gophercloud.EndpointOpts{
		Name:   "neutron",
		Region: cluster.Spec.Cloud.Region,
	})
	if err != nil {
		return nil, err
	}
	return &cloudConnector{
		ctx:           ctx,
		cluster:       cluster,
		namer:         namer{cluster},
		computeClient: computeClient,
		networkClient: networkClient,
	}, nil
}

func (conn *cloudConnector) detectInstanceImage() error {
	opts := images.ListOpts{ChangesSince: "2014-01-01T01:02:03Z", Name: "Ubuntu 16.04"}
	pager := images.ListDetail(conn.computeClient, opts)
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

func (conn *cloudConnector) getFlavorRef(name string) (string, error) {
	opts := flavors.ListOpts{ChangesSince: "2014-01-01T01:02:03Z", MinRAM: 1}
	pager := flavors.ListDetail(conn.computeClient, opts)
	flavourRef := ""
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		flavorList, err := flavors.ExtractFlavors(page)
		if err != nil {
			return false, err
		}

		for _, f := range flavorList {
			// "f" will be a flavors.Flavor
			if f.Name == name {
				flavourRef = f.ID
				return true, nil
			}
		}
		return false, nil
	})
	return flavourRef, err
}

func (conn *cloudConnector) getSecurityGroup(name string) (*secgroups.SecurityGroup, error) {
	pager := secgroups.List(conn.computeClient)
	var group *secgroups.SecurityGroup
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		groupList, err := secgroups.ExtractSecurityGroups(page)
		if err != nil {
			return false, err
		}

		for _, g := range groupList {
			// "f" will be a flavors.Flavor
			if g.Name == name {
				group = &g
				return true, nil
			}
		}
		return false, nil
	})

	return group, err
}

func (conn *cloudConnector) CreateSecurityGroup() error {
	group, err := conn.getSecurityGroup(conn.namer.GetSecurityGroupName())
	if err != nil {
		return err
	}
	if group.ID == "" {
		opts := secgroups.CreateOpts{
			Name:        conn.namer.GetSecurityGroupName(),
			Description: "pharmer default api port address",
		}
		group, err = secgroups.Create(conn.computeClient, opts).Extract()
		if err != nil {
			return err
		}
	}

	ruleOpts := secgroups.CreateRuleOpts{
		ParentGroupID: group.ID,
		FromPort:      6443,
		ToPort:        6443,
		IPProtocol:    "TCP",
		CIDR:          "0.0.0.0/0",
	}
	return secgroups.CreateRule(conn.computeClient, ruleOpts).Err

}

func (conn *cloudConnector) getSharedNetwork() (string, error) {
	opts := networks.ListOpts{Shared: BoolP(true)}
	pager := networks.List(conn.networkClient, opts)
	networkUID := ""
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		networkList, err := networks.ExtractNetworks(page)
		if err != nil {
			return false, err
		}
		for _, n := range networkList {
			if n.Status == "ACTIVE" {
				networkUID = n.ID
				return true, nil
			}
		}
		return false, nil
	})
	return networkUID, err
}

func (conn *cloudConnector) createNetwork() (string, error) {
	return "", nil
}

func (conn *cloudConnector) CreateInstance(name, token string, ng *api.NodeGroup) (*api.NodeInfo, error) {
	flavorRef, err := conn.getFlavorRef(ng.Spec.Template.Spec.SKU)
	if err != nil {
		return nil, err
	}
	networkUid, err := conn.getSharedNetwork()
	if err != nil {
		return nil, err
	}

	script, err := conn.renderStartupScript(ng, token)

	opts := servers.CreateOpts{
		Name:      name,
		ImageRef:  conn.cluster.Spec.Cloud.InstanceImage,
		FlavorRef: flavorRef,
		Networks: []servers.Network{
			{
				UUID: networkUid,
			},
		},
		UserData: []byte(script),
	}
	createOpts := keypairs.CreateOptsExt{
		CreateOptsBuilder: opts,
		KeyName:           conn.cluster.Status.Cloud.SShKeyExternalID,
	}

	server, err := servers.Create(conn.computeClient, createOpts).Extract()
	if err != nil {
		return nil, err
	}

	node := api.NodeInfo{
		Name:       server.Name,
		ExternalID: server.ID,
	}

	err = servers.ListAddresses(conn.computeClient, server.ID).EachPage(func(page pagination.Page) (bool, error) {
		as, err := servers.ExtractAddresses(page)
		if err != nil {
			return false, err
		}
		for key, a := range as {
			if key == "VLAN" {
				node.PrivateIP = a[0].Address
			} else {
				node.PublicIP = a[0].Address
			}
		}
		return true, nil

	})
	if err != nil {
		return nil, err
	}

	return &node, nil
}

func (conn *cloudConnector) DeleteInstanceByProviderID(providerID string) error {
	return nil
}

func (conn *cloudConnector) getPublicKey() (bool, string, error) {
	pager := keypairs.List(conn.computeClient)
	keyID := ""
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		keyList, err := keypairs.ExtractKeyPairs(page)
		if err != nil {
			return false, err
		}
		for _, k := range keyList {
			if k.Name == conn.cluster.Spec.Cloud.SSHKeyName {
				keyID = k.Name
				return true, nil
			}
		}
		return false, nil
	})
	return keyID != "", keyID, err
}

func (conn *cloudConnector) deleteSSHKey(sshKeyName string) error {
	return keypairs.Delete(conn.computeClient, sshKeyName).ExtractErr()
}

func (conn *cloudConnector) importPublicKey() error {
	Logger(conn.ctx).Infof("Adding SSH public key")
	opts := keypairs.CreateOpts{
		Name:      conn.cluster.Spec.Cloud.SSHKeyName,
		PublicKey: string(SSHKey(conn.ctx).PublicKey),
	}
	resp, err := keypairs.Create(conn.computeClient, opts).Extract()
	if err != nil {
		return errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	conn.cluster.Status.Cloud.SShKeyExternalID = resp.Name
	Logger(conn.ctx).Infof("New ssh key with name %v and id %v created", conn.cluster.Spec.Cloud.SSHKeyName, resp.Name)
	return nil
}
