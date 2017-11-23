package packet

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/appscode/go/errors"
	api "github.com/appscode/pharmer/apis/v1alpha1"
	. "github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/credential"
	"github.com/packethost/packngo"
	"k8s.io/apimachinery/pkg/util/wait"
)

type cloudConnector struct {
	ctx     context.Context
	cluster *api.Cluster
	client  *packngo.Client
}

func NewConnector(ctx context.Context, cluster *api.Cluster) (*cloudConnector, error) {
	cred, err := Store(ctx).Credentials().Get(cluster.Spec.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.Packet{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.New().WithMessagef("Credential %s is invalid. Reason: %v", cluster.Spec.CredentialName, err)
	}
	// TODO: FixIt Project ID
	cluster.Spec.Cloud.Project = typed.ProjectID()

	return &cloudConnector{
		ctx:     ctx,
		cluster: cluster,
		client:  packngo.NewClient("", typed.APIKey(), nil),
	}, nil
}

func (conn *cloudConnector) waitForInstance(deviceID, status string) error {
	attempt := 0
	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		attempt++

		server, _, err := conn.client.Devices.Get(deviceID)
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
	if conn.cluster.Status.Cloud.SShKeyExternalID == "" {
		return false, "", nil
	}
	key, resp, err := conn.client.SSHKeys.Get(conn.cluster.Status.Cloud.SShKeyExternalID)
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	return true, key.ID, nil
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
		Facility:     conn.cluster.Spec.Cloud.Zone,
		OS:           conn.cluster.Spec.Cloud.InstanceImage,
		BillingCycle: "hourly",
		ProjectID:    conn.cluster.Spec.Cloud.Project,
		UserData:     script,
		Tags:         []string{conn.cluster.Name},
		SpotInstance: ng.Spec.Template.Spec.SpotInstances,
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

	host, _, err := conn.client.Devices.Get(server.ID)
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
		return "", fmt.Errorf("unexpected providerID format: %s, format should be: digitalocean://12345", providerID)
	}

	// since split[0] is actually "digitalocean:"
	if strings.TrimSuffix(split[0], ":") != UID {
		return "", fmt.Errorf("provider name from providerID should be digitalocean: %s", providerID)
	}

	return split[2], nil
}

// ---------------------------------------------------------------------------------------------------------------------

// reboot does not seem to run /etc/rc.local
func (conn *cloudConnector) reboot(id string) error {
	Logger(conn.ctx).Infof("Rebooting instance %v", id)
	_, err := conn.client.Devices.Reboot(id)
	if err != nil {
		return errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	return nil
}
