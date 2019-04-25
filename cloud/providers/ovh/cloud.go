package ovh

import (
	"context"
	"strings"

	. "github.com/appscode/go/context"
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
	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/credential"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/wait"
)

const auth_url = "https://auth.cloud.ovh.net/v2.0"

type cloudConnector struct {
	ctx           context.Context
	cluster       *api.Cluster
	computeClient *gophercloud.ServiceClient
	networkClient *gophercloud.ServiceClient
	namer         namer
}

var _ InstanceManager = &cloudConnector{}

func NewConnector(ctx context.Context, cluster *api.Cluster) (*cloudConnector, error) {
	cred, err := Store(ctx).Owner(owner).Credentials().Get(cluster.Spec.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.Ovh{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.Wrapf(err, "credential %s is invalid", cluster.Spec.CredentialName)
	}

	opts := gophercloud.AuthOptions{
		IdentityEndpoint: auth_url,
		Username:         typed.Username(),
		Password:         typed.Password(),
		TenantID:         typed.TenantID(),
	}

	provider, err := openstack.AuthenticatedClient(opts)
	if err != nil {
		return nil, errors.Wrapf(err, "credential %s is not authenticated", cluster.Spec.CredentialName)
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

func (conn *cloudConnector) getNetwork(isShared bool) (string, error) {
	opts := networks.ListOpts{Shared: BoolP(isShared)}
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

func (conn *cloudConnector) CreateInstance(name, token string, ng *api.NodeGroup) (*api.NodeInfo, error) {
	flavorRef, err := conn.getFlavorRef(ng.Spec.Template.Spec.SKU)
	if err != nil {
		return nil, err
	}
	publicNetworkUid, err := conn.getNetwork(true)
	if err != nil {
		return nil, err
	}

	privateNetworkUid, err := conn.getNetwork(false)
	if err != nil {
		return nil, err
	}

	script, err := conn.renderStartupScript(ng, token)
	if err != nil {
		return nil, err
	}

	opts := servers.CreateOpts{
		Name:      name,
		ImageRef:  conn.cluster.Spec.Cloud.InstanceImage,
		FlavorRef: flavorRef,
		Networks: []servers.Network{
			{
				UUID: publicNetworkUid,
			},
			{
				UUID: privateNetworkUid,
			},
		},
		UserData: []byte(script),
	}
	createOpts := keypairs.CreateOptsExt{
		CreateOptsBuilder: opts,
		KeyName:           conn.cluster.Spec.Cloud.SSHKeyName,
	}

	server, err := servers.Create(conn.computeClient, createOpts).Extract()
	if err != nil {
		return nil, err
	}

	Logger(conn.ctx).Infof("Ovh server %v created", name)

	if err = conn.waitForActiveInstance(server.ID); err != nil {
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

func (conn *cloudConnector) waitForActiveInstance(id string) error {
	attempt := 0
	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		attempt++

		server, err := servers.Get(conn.computeClient, id).Extract()
		if err != nil {
			return false, nil
		}
		Logger(conn.ctx).Infof("Attempt %v: Instance `%v` is in status `%s`", attempt, id, server.Status)
		if strings.ToLower(server.Status) == "active" {
			return true, nil
		}
		return false, nil
	})
}

func (conn *cloudConnector) DeleteInstanceByProviderID(providerID string) error {
	serverId, err := serverIDFromProviderID(providerID)
	if err != nil {
		return err
	}
	return servers.Delete(conn.computeClient, serverId).ExtractErr()
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
		return errors.Wrap(err, ID(conn.ctx))
	}
	conn.cluster.Status.Cloud.SShKeyExternalID = resp.Name
	Logger(conn.ctx).Infof("New ssh key with name %v and id %v created", conn.cluster.Spec.Cloud.SSHKeyName, resp.Name)
	return nil
}

func serverIDFromProviderID(providerID string) (string, error) {
	if providerID == "" {
		return "", errors.New("providerID cannot be empty string")
	}

	split := strings.Split(providerID, "/")
	if len(split) != 4 {
		return "", errors.Errorf("unexpected providerID format: %s, format should be: pharmer-openstack:///12345", providerID)
	}

	return split[3], nil
}
