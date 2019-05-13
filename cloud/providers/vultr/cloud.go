package vultr

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	gv "github.com/JamesClonk/vultr/lib"
	. "github.com/appscode/go/context"
	"github.com/pharmer/cloud/pkg/credential"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/apis/v1beta1/vultr"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

type cloudConnector struct {
	ctx     context.Context
	cluster *api.Cluster
	client  *gv.Client
	namer   namer
}

//var _ InstanceManager = &cloudConnector{}

func NewConnector(ctx context.Context, cluster *api.Cluster, owner string) (*cloudConnector, error) {
	cred, err := Store(ctx).Owner(owner).Credentials().Get(cluster.ClusterConfig().CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.Vultr{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.Wrapf(err, "credential %s is invalid", cluster.ClusterConfig().CredentialName)
	}

	return &cloudConnector{
		ctx:     ctx,
		cluster: cluster,
		namer:   namer{cluster},
		client:  gv.NewClient(typed.Token(), nil),
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

func (conn *cloudConnector) detectInstanceImage(owner string) error {
	oses, err := conn.client.GetOS()
	if err != nil {
		return errors.Wrap(err, ID(conn.ctx))
	}
	for _, os := range oses {
		if os.Arch == "x64" && os.Family == "ubuntu" && strings.HasPrefix(os.Name, "Ubuntu 16.04 x64") {
			conn.cluster.ClusterConfig().Cloud.InstanceImage = strconv.Itoa(os.ID)
			return nil
		}
	}

	return errors.Errorf("[%s] can't find Debian 8 image", ID(conn.ctx))
}

/*
The "status" field represents the status of the subscription and will be one of:
pending | active | suspended | closed. If the status is "active", you can check "power_status"
to determine if the VPS is powered on or not. When status is "active", you may also use
"server_state" for a more detailed status of: none | locked | installingbooting | isomounting | ok.
*/
func (conn *cloudConnector) waitForActiveInstance(id string) error {
	attempt := 0
	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		attempt++

		server, err := conn.client.GetServer(id)
		if err != nil {
			return false, nil
		}
		Logger(conn.ctx).Infof("Attempt %v: Instance `%v` is in status `%s`", attempt, id, server.Status)
		if strings.ToLower(server.Status) == "active" && server.PowerStatus == "running" {
			return true, nil
		}
		return false, nil
	})
}

// ---------------------------------------------------------------------------------------------------------------------

func (conn *cloudConnector) getPublicKey() (bool, string, error) {
	keys, err := conn.client.GetSSHKeys()
	if err != nil {
		return false, "", err
	}
	for _, key := range keys {
		if key.Name == conn.cluster.ClusterConfig().Cloud.SSHKeyName {
			return true, key.ID, nil
		}
	}
	return false, "", nil
}

func (conn *cloudConnector) importPublicKey() error {
	Logger(conn.ctx).Infof("Adding SSH public key")
	resp, err := conn.client.CreateSSHKey(conn.cluster.ClusterConfig().Cloud.SSHKeyName, string(SSHKey(conn.ctx).PublicKey))
	if err != nil {
		return errors.Wrap(err, ID(conn.ctx))
	}
	conn.cluster.Status.Cloud.SShKeyExternalID = resp.ID
	Logger(conn.ctx).Infof("New ssh key with name %v and id %v created", conn.cluster.ClusterConfig().Cloud.SSHKeyName, resp.ID)
	return nil
}

func (conn *cloudConnector) deleteSSHKey(id string) error {
	Logger(conn.ctx).Infof("Deleting SSH key for cluster %s", conn.cluster.Name)
	err := wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		err := conn.client.DeleteSSHKey(id)
		return err == nil, nil
	})
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("SSH key for cluster %v deleted", conn.cluster.Name)
	return nil
}

// ---------------------------------------------------------------------------------------------------------------------

func (conn *cloudConnector) getReserveIP(ip string) (bool, error) {
	ips, err := conn.client.ListReservedIP()
	if err != nil {
		return false, err
	}
	for _, data := range ips {
		if data.Subnet == ip {
			return true, nil
		}
	}
	return false, nil
}

func (conn *cloudConnector) createReserveIP() (string, error) {
	regionID, err := strconv.Atoi(conn.cluster.ClusterConfig().Cloud.Zone)
	if err != nil {
		return "", err
	}
	ipID, err := conn.client.CreateReservedIP(regionID, "v4", conn.namer.ReserveIPName())
	if err != nil {
		return "", err
	}
	Logger(conn.ctx).Infof("Reserved new floating IP=%v", ipID)

	ip, err := conn.client.GetReservedIP(ipID)
	if err != nil {
		return "", err
	}
	Logger(conn.ctx).Infof("Floating ip %v reserved", ip.Subnet)
	return ip.Subnet, nil
}

func (conn *cloudConnector) assignReservedIP(ip, serverId string) error {
	err := conn.client.AttachReservedIP(ip, serverId)
	if err != nil {
		return errors.Wrap(err, ID(conn.ctx))
	}
	Logger(conn.ctx).Infof("Reserved ip %v assigned to %v", ip, serverId)
	return nil
}

func (conn *cloudConnector) releaseReservedIP(ip string) error {
	Logger(conn.ctx).Debugln("Deleting Floating IP", ip)
	err := conn.client.DestroyReservedIP(ip)
	if err != nil {
		return errors.WithStack(err)
	}
	Logger(conn.ctx).Infof("Reserved IP %v deleted", ip)
	return nil
}

// ---------------------------------------------------------------------------------------------------------------------

func (conn *cloudConnector) getStartupScriptID(machine *clusterv1.Machine) (int, error) {
	machineConfig, err := vultr.MachineConfigFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		return 0, err
	}

	scriptName := conn.namer.StartupScriptName(machine.Name, string(machineConfig.Roles[0]))

	scripts, err := conn.client.GetStartupScripts()
	if err != nil {
		return 0, err
	}
	for _, script := range scripts {
		if script.Name == scriptName {
			scriptID, err := strconv.Atoi(script.ID)
			if err != nil {
				return 0, err
			}
			return scriptID, nil
		}
	}
	return 0, ErrNotFound
}

func (conn *cloudConnector) createOrUpdateStartupScript(machine *clusterv1.Machine, token string, owner string) (int, error) {
	machineConfig, err := vultr.MachineConfigFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		return 0, err
	}
	scriptName := conn.namer.StartupScriptName(machine.Name, string(machineConfig.Roles[0]))
	script, err := conn.renderStartupScript(conn.cluster, machine, token, owner)
	if err != nil {
		return 0, err
	}

	scripts, err := conn.client.GetStartupScripts()
	if err != nil {
		return 0, err
	}
	for _, s := range scripts {
		if s.Name == scriptName {
			s.Content = script
			err := conn.client.UpdateStartupScript(s)
			if err != nil {
				return 0, err
			}
			return strconv.Atoi(s.ID)
		}
	}

	Logger(conn.ctx).Infof("creating StackScript for NodeGroup %v role %v", machine.Name, machineConfig.Roles[0])
	resp, err := conn.client.CreateStartupScript(scriptName, script, "boot")
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(resp.ID)
}

func (conn *cloudConnector) deleteStartupScript(name string, roles string) error {
	scriptName := conn.namer.StartupScriptName(name, roles)
	scripts, err := conn.client.GetStartupScripts()
	if err != nil {
		return err
	}
	for _, script := range scripts {
		if script.Name == scriptName {
			return conn.client.DeleteStartupScript(script.ID)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------------------------------------------------

func (conn *cloudConnector) CreateInstance(name, token string, machine *clusterv1.Machine, owner string) (*api.NodeInfo, error) {
	regionID, err := strconv.Atoi(conn.cluster.ClusterConfig().Cloud.Zone)
	if err != nil {
		return nil, err
	}
	machineConfig, err := vultr.MachineConfigFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		return nil, err
	}
	planID, err := strconv.Atoi(machineConfig.Plan)
	if err != nil {
		return nil, err
	}
	osID, err := strconv.Atoi(conn.cluster.ClusterConfig().Cloud.InstanceImage)
	if err != nil {
		return nil, err
	}

	_, sshKeyID, err := conn.getPublicKey()
	if err != nil {
		return nil, err
	}
	_, err = conn.createOrUpdateStartupScript(machine, token, owner)
	if err != nil {
		return nil, err
	}

	scriptID, err := conn.getStartupScriptID(machine)
	if err != nil {
		return nil, err
	}

	opts := &gv.ServerOptions{
		SSHKey:               sshKeyID + ",57dcbce7cd3b6,58027d56a1190,58a498ec7ee19,57ee2df762851",
		PrivateNetworking:    true,
		DontNotifyOnActivate: false,
		Script:               scriptID,
		Hostname:             name,
		Tag:                  conn.cluster.Name,
	}
	if Env(conn.ctx).IsPublic() {
		opts.SSHKey = sshKeyID
	}
	resp, err := conn.client.CreateServer(name, regionID, planID, osID, opts)
	if err != nil {
		return nil, err
	}
	Logger(conn.ctx).Infof("Vultr server %v created", name)

	if err = conn.waitForActiveInstance(resp.ID); err != nil {
		return nil, err
	}

	// load again to get IP address assigned
	host, err := conn.client.GetServer(resp.ID)
	if err != nil {
		return nil, errors.Wrap(err, ID(conn.ctx))
	}
	node := api.NodeInfo{
		Name:       host.Name,
		ExternalID: host.ID,
		PublicIP:   host.MainIP,
		PrivateIP:  host.InternalIP,
	}
	return &node, nil
}

func (conn *cloudConnector) DeleteInstanceByProviderID(providerID string) error {
	dropletID, err := serverIDFromProviderID(providerID)
	if err != nil {
		return err
	}
	err = conn.client.DeleteServer(dropletID)
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("Droplet %v deleted", dropletID)
	return nil
}

// serverIDFromProviderID returns a server's ID from providerID.
//
// The providerID spec should be retrievable from the Kubernetes
// node object. The expected format is: vultr://server-id
func serverIDFromProviderID(providerID string) (string, error) {
	if providerID == "" {
		return "", errors.New("providerID cannot be empty string")
	}

	split := strings.Split(providerID, "/")
	if len(split) != 3 {
		return "", errors.Errorf("unexpected providerID format: %s, format should be: vultr://12345", providerID)
	}

	// since split[0] is actually "digitalocean:"
	if strings.TrimSuffix(split[0], ":") != UID {
		return "", errors.Errorf("provider name from providerID should be vultr: %s", providerID)
	}

	return split[2], nil
}

func (conn *cloudConnector) instanceIfExists(machine *clusterv1.Machine) (*gv.Server, error) {
	servers, err := conn.client.GetServers()
	if err != nil {
		return nil, err
	}
	for _, server := range servers {
		if server.Name == machine.Name {
			return &server, nil
		}
	}

	return nil, fmt.Errorf("no server found with %v name", machine.Name)
}

func (conn *cloudConnector) CreateCredentialSecret(kc kubernetes.Interface, data map[string]string) error {
	return nil
}

// ---------------------------------------------------------------------------------------------------------------------
