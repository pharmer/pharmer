package packet

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	. "github.com/appscode/go/context"
	"github.com/packethost/packngo"
	"github.com/pharmer/cloud/pkg/credential"
	"github.com/pharmer/cloud/pkg/providers"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

type cloudConnector struct {
	ctx     context.Context
	i       providers.Interface
	cluster *api.Cluster
	client  *packngo.Client
}

func NewConnector(ctx context.Context, cluster *api.Cluster, owner string) (*cloudConnector, error) {
	cred, err := Store(ctx).Owner(owner).Credentials().Get(cluster.ClusterConfig().CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.Packet{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.Wrapf(err, "credential %s is invalid", cluster.ClusterConfig().CredentialName)
	}
	// TODO: FixIt Project ID
	cluster.ClusterConfig().Cloud.Project = typed.ProjectID()

	i, err := providers.NewCloudProvider(providers.Options{
		Provider: cluster.Spec.Config.Cloud.CloudProvider,
		// set credentials
	})
	if err != nil {
		return nil, err
	}

	return &cloudConnector{
		ctx:     ctx,
		i:       i,
		cluster: cluster,
		client:  packngo.NewClientWithAuth("", typed.APIKey(), nil),
	}, nil
}

func PrepareCloud(ctx context.Context, clusterName string, owner string) (*cloudConnector, error) {
	var err error
	var conn *cloudConnector
	cluster, err := Store(ctx).Owner(owner).Clusters().Get(clusterName)
	if err != nil {
		return conn, fmt.Errorf("cluster `%s` does not exist. Reason: %v", clusterName, err)
	}

	if ctx, err = LoadCACertificates(ctx, cluster, owner); err != nil {
		return conn, err
	}

	if ctx, err = LoadSSHKey(ctx, cluster, owner); err != nil {
		return conn, err
	}
	if conn, err = NewConnector(ctx, cluster, owner); err != nil {
		return nil, err
	}
	return conn, nil
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

	keys, _, err := conn.client.SSHKeys.ProjectList(conn.cluster.ClusterConfig().Cloud.Project)
	for _, key := range keys {
		if key.Label == conn.cluster.ClusterConfig().Cloud.SSHKeyName {
			return true, key.ID, nil
		}
	}
	return false, "", err
}

func (conn *cloudConnector) importPublicKey() (string, error) {
	Logger(conn.ctx).Debugln("Adding SSH public key")
	sk, _, err := conn.client.SSHKeys.Create(&packngo.SSHKeyCreateRequest{
		Key:       string(SSHKey(conn.ctx).PublicKey),
		Label:     conn.cluster.ClusterConfig().Cloud.SSHKeyName,
		ProjectID: conn.cluster.ClusterConfig().Cloud.Project,
	})

	if err != nil {
		found, keyID, err := conn.getPublicKey()
		if !found {
			return "", err
		}
		return keyID, err
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

func (conn *cloudConnector) CreateInstance(machine *clusterv1.Machine, token, owner string) (*api.NodeInfo, error) {
	script, err := conn.renderStartupScript(conn.cluster, machine, token)
	if err != nil {
		return nil, err
	}
	machineConfig, err := machineProviderFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		return nil, err
	}
	server, _, err := conn.client.Devices.Create(&packngo.DeviceCreateRequest{
		Hostname:     machine.Name,
		Plan:         machineConfig.Plan,
		Facility:     []string{conn.cluster.ClusterConfig().Cloud.Zone},
		OS:           conn.cluster.ClusterConfig().Cloud.InstanceImage,
		BillingCycle: "hourly",
		ProjectID:    conn.cluster.ClusterConfig().Cloud.Project,
		UserData:     script,
		Tags:         []string{conn.cluster.Name},
		SpotInstance: api.NodeType(machineConfig.SpotInstance) == api.NodeTypeSpot,
	})
	if err != nil {
		return nil, err
	}
	Logger(conn.ctx).Infof("Instance %v created", machine.Name)

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
	serverID, err := serverIDFromProviderID(providerID)
	if err != nil {
		return err
	}
	_, err = conn.client.Devices.Delete(serverID)
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("Server %v deleted", serverID)
	return nil
}

func serverIDFromProviderID(providerID string) (string, error) {
	if providerID == "" {
		return "", errors.New("providerID cannot be empty string")
	}

	split := strings.Split(providerID, "/")
	if len(split) != 3 {
		return "", errors.Errorf("unexpected providerID format: %s, format should be: packet://12345", providerID)
	}

	// since split[0] is actually "packet:"
	if strings.TrimSuffix(split[0], ":") != UID {
		return "", errors.Errorf("provider name from providerID should be packet: %s", providerID)
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

func (conn *cloudConnector) instanceIfExists(machine *clusterv1.Machine) (*packngo.Device, error) {
	devices, _, err := conn.client.Devices.List(conn.cluster.ClusterConfig().Cloud.Project, nil)
	if err != nil {
		return nil, err
	}
	for _, device := range devices {
		if device.Hostname == machine.Name {
			return &device, nil
		}
	}
	return nil, fmt.Errorf("no server found with %v name", machine.Name)
}

func (conn *cloudConnector) CreateCredentialSecret(kc kubernetes.Interface, data map[string]string) error {
	return nil
}
