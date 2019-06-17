package packet

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/appscode/go/log"
	"github.com/packethost/packngo"
	"github.com/pharmer/cloud/pkg/credential"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

type cloudConnector struct {
	*CloudManager
	client *packngo.Client
}

func NewConnector(cm *ClusterManager) (*cloudConnector, error) {
	cluster := cm.Cluster

	cred, err := store.StoreProvider.Credentials().Get(cluster.ClusterConfig().CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.Packet{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.Wrapf(err, "credential %s is invalid", cluster.ClusterConfig().CredentialName)
	}
	// TODO: FixIt Project ID
	cluster.Spec.Config.Cloud.Project = typed.ProjectID()

	return &cloudConnector{
		CloudManager: cm.CloudManager,
		client:       packngo.NewClientWithAuth("", typed.APIKey(), nil),
	}, nil
}

func (conn *cloudConnector) waitForInstance(deviceID, status string) error {
	attempt := 0
	return wait.PollImmediate(api.RetryInterval, api.RetryTimeout, func() (bool, error) {
		attempt++

		server, _, err := conn.client.Devices.Get(deviceID, &packngo.GetOptions{})
		if err != nil {
			return false, nil
		}
		log.Infof("Attempt %v: Instance `%v` is in status `%s`", attempt, server.ID, server.State)
		if strings.ToLower(server.State) == status {
			return true, nil
		}
		return false, nil
	})
}

func (conn *cloudConnector) getPublicKey() (bool, string, error) {
	if conn.Cluster.Status.Cloud.SShKeyExternalID != "" {
		key, resp, err := conn.client.SSHKeys.Get(conn.Cluster.Status.Cloud.SShKeyExternalID, &packngo.GetOptions{})
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return false, "", nil
		}
		if err != nil {
			return false, "", err
		}
		return true, key.ID, nil
	}

	keys, _, err := conn.client.SSHKeys.ProjectList(conn.Cluster.ClusterConfig().Cloud.Project)
	for _, key := range keys {
		if key.Label == conn.Cluster.ClusterConfig().Cloud.SSHKeyName {
			return true, key.ID, nil
		}
	}
	return false, "", err
}

func (conn *cloudConnector) importPublicKey() (string, error) {
	log.Debugln("Adding SSH public key")
	sk, _, err := conn.client.SSHKeys.Create(&packngo.SSHKeyCreateRequest{
		Key:       string(conn.Certs.SSHKey.PublicKey),
		Label:     conn.Cluster.ClusterConfig().Cloud.SSHKeyName,
		ProjectID: conn.Cluster.ClusterConfig().Cloud.Project,
	})

	if err != nil {
		found, keyID, err := conn.getPublicKey()
		if !found {
			return "", err
		}
		return keyID, err
	}
	log.Debugf("Created new ssh key with fingerprint=%v", conn.Certs.SSHKey.OpensshFingerprint)
	return sk.ID, nil
}

func (conn *cloudConnector) CreateInstance(machine *clusterv1.Machine, script string) (*api.NodeInfo, error) {
	machineConfig, err := machineProviderFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		return nil, err
	}
	server, _, err := conn.client.Devices.Create(&packngo.DeviceCreateRequest{
		Hostname:     machine.Name,
		Plan:         machineConfig.Plan,
		Facility:     []string{conn.Cluster.ClusterConfig().Cloud.Zone},
		OS:           conn.Cluster.ClusterConfig().Cloud.InstanceImage,
		BillingCycle: "hourly",
		ProjectID:    conn.Cluster.ClusterConfig().Cloud.Project,
		UserData:     script,
		Tags:         []string{conn.Cluster.Name},
		SpotInstance: api.NodeType(machineConfig.SpotInstance) == api.NodeTypeSpot,
	})
	if err != nil {
		return nil, err
	}
	log.Infof("Instance %v created", machine.Name)

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
	log.Infof("Server %v deleted", serverID)
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

func (conn *cloudConnector) instanceIfExists(machine *clusterv1.Machine) (*packngo.Device, error) {
	devices, _, err := conn.client.Devices.List(conn.Cluster.ClusterConfig().Cloud.Project, nil)
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
