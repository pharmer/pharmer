package packet

import (
	"context"
	"net/http"
	"strings"

	. "github.com/appscode/go/context"
	"github.com/packethost/packngo"
	api "github.com/pharmer/pharmer/apis/v1alpha1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/credential"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/wait"
)

type cloudConnector struct {
	ctx     context.Context
	cluster *api.Cluster
	client  *packngo.Client
}

func NewConnector(ctx context.Context, cluster *api.Cluster) (*cloudConnector, error) {
	cred, err := Store(ctx).Owner(owner).Credentials().Get(cluster.Spec.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.Packet{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.Wrapf(err, "credential %s is invalid", cluster.Spec.CredentialName)
	}
	// TODO: FixIt Project ID
	cluster.Spec.Cloud.Project = typed.ProjectID()

	return &cloudConnector{
		ctx:     ctx,
		cluster: cluster,
		client:  packngo.NewClientWithAuth("", typed.APIKey(), nil),
	}, nil
}

func (conn *cloudConnector) waitForInstance(deviceID, status string) error {
	attempt := 0
	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		attempt++

		server, _, err := conn.client.Devices.Get(deviceID, &packngo.GetOptions{})
		if err != nil {
			return false, nil
		}
		Logger(conn.ctx).Infof("Attempt %v: Instance `%v` is in status `%s`", attempt, server.ID, server.State)
		if strings.ToLower(server.State) == status {
			return true, nil
		}
		return false, nil
	})
}

// ---------------------------------------------------------------------------------------------------------------------

func (conn *cloudConnector) getPublicKey() (bool, string, error) {
	if conn.cluster.Status.Cloud.SShKeyExternalID != "" {
		key, resp, err := conn.client.SSHKeys.Get(conn.cluster.Status.Cloud.SShKeyExternalID, &packngo.GetOptions{})
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return false, "", nil
		}
		if err != nil {
			return false, "", err
		}
		return true, key.ID, nil
	}

	keys, _, err := conn.client.SSHKeys.ProjectList(conn.cluster.Spec.Cloud.Project)
	for _, key := range keys {
		if key.Label == conn.cluster.Spec.Cloud.SSHKeyName {
			return true, key.ID, nil
		}
	}
	return false, "", err
}

func (conn *cloudConnector) importPublicKey() (string, error) {
	Logger(conn.ctx).Debugln("Adding SSH public key")
	sk, _, err := conn.client.SSHKeys.Create(&packngo.SSHKeyCreateRequest{
		Key:       string(SSHKey(conn.ctx).PublicKey),
		Label:     conn.cluster.Spec.Cloud.SSHKeyName,
		ProjectID: conn.cluster.Spec.Cloud.Project,
	})
	if err != nil {
		return "", err
	}
	Logger(conn.ctx).Debugf("Created new ssh key with fingerprint=%v", SSHKey(conn.ctx).OpensshFingerprint)
	return sk.ID, nil
}

func (conn *cloudConnector) deleteSSHKey(id string) error {
	Logger(conn.ctx).Infof("Deleting SSH key for cluster %s", conn.cluster.Name)
	return wait.PollImmediate(RetryInterval, RetryInterval, func() (bool, error) {
		_, err := conn.client.SSHKeys.Delete(id)
		return err == nil, nil
	})
}

// ---------------------------------------------------------------------------------------------------------------------

func (conn *cloudConnector) CreateInstance(name, token string, ng *api.NodeGroup) (*api.NodeInfo, error) {
	script, err := conn.renderStartupScript(ng, token)
	if err != nil {
		return nil, err
	}

	server, _, err := conn.client.Devices.Create(&packngo.DeviceCreateRequest{
		Hostname:     name,
		Plan:         ng.Spec.Template.Spec.SKU,
		Facility:     []string{conn.cluster.Spec.Cloud.Zone},
		OS:           conn.cluster.Spec.Cloud.InstanceImage,
		BillingCycle: "hourly",
		ProjectID:    conn.cluster.Spec.Cloud.Project,
		UserData:     script,
		Tags:         []string{conn.cluster.Name},
		SpotInstance: ng.Spec.Template.Spec.Type == api.NodeTypeSpot,
		SpotPriceMax: ng.Spec.Template.Spec.SpotPriceMax,
	})
	if err != nil {
		return nil, err
	}
	Logger(conn.ctx).Infof("Instance %v created", name)

	err = conn.waitForInstance(server.ID, "active")
	if err != nil {
		return nil, err
	}

	host, _, err := conn.client.Devices.Get(server.ID, &packngo.GetOptions{})
	if err != nil {
		return nil, err
	}
	node := api.NodeInfo{
		Name:       host.Hostname,
		ExternalID: host.ID,
	}
	for _, addr := range host.Network {
		if addr.AddressFamily == 4 {
			if addr.Public {
				node.PublicIP = addr.Address
			} else {
				node.PrivateIP = addr.Address
			}
		}
	}
	return &node, nil
}

func (conn *cloudConnector) DeleteInstanceByProviderID(providerID string) error {
	dropletID, err := serverIDFromProviderID(providerID)
	if err != nil {
		return err
	}
	_, err = conn.client.Devices.Delete(dropletID)
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("Droplet %v deleted", dropletID)
	return nil
}

// dropletIDFromProviderID returns a droplet's ID from providerID.
//
// The providerID spec should be retrievable from the Kubernetes
// node object. The expected format is: digitalocean://droplet-id
// ref: https://github.com/digitalocean/digitalocean-cloud-controller-manager/blob/f9a9856e99c9d382db3777d678f29d85dea25e91/do/droplets.go#L211
func serverIDFromProviderID(providerID string) (string, error) {
	if providerID == "" {
		return "", errors.New("providerID cannot be empty string")
	}

	split := strings.Split(providerID, "/")
	if len(split) != 3 {
		return "", errors.Errorf("unexpected providerID format: %s, format should be: digitalocean://12345", providerID)
	}

	// since split[0] is actually "digitalocean:"
	if strings.TrimSuffix(split[0], ":") != UID {
		return "", errors.Errorf("provider name from providerID should be digitalocean: %s", providerID)
	}

	return split[2], nil
}

// ---------------------------------------------------------------------------------------------------------------------

// reboot does not seem to run /etc/rc.local
func (conn *cloudConnector) reboot(id string) error {
	Logger(conn.ctx).Infof("Rebooting instance %v", id)
	_, err := conn.client.Devices.Reboot(id)
	if err != nil {
		return errors.Wrap(err, ID(conn.ctx))
	}
	return nil
}
