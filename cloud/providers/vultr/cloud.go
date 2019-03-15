package vultr

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	gv "github.com/JamesClonk/vultr/lib"
	. "github.com/appscode/go/context"
	api "github.com/pharmer/pharmer/apis/v1alpha1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/credential"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/wait"
)

type cloudConnector struct {
	ctx     context.Context
	cluster *api.Cluster
	client  *gv.Client
	namer   namer
}

var _ InstanceManager = &cloudConnector{}

func NewConnector(ctx context.Context, cluster *api.Cluster) (*cloudConnector, error) {
	cred, err := Store(ctx).Owner(owner).Credentials().Get(cluster.Spec.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.Vultr{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.Wrapf(err, "credential %s is invalid", cluster.Spec.CredentialName)
	}

	return &cloudConnector{
		ctx:     ctx,
		cluster: cluster,
		namer:   namer{cluster},
		client:  gv.NewClient(typed.Token(), nil),
	}, nil
}

func (conn *cloudConnector) detectInstanceImage() error {
	oses, err := conn.client.GetOS()
	if err != nil {
		return errors.Wrap(err, ID(conn.ctx))
	}
	for _, os := range oses {
		if os.Arch == "x64" && os.Family == "ubuntu" && strings.HasPrefix(os.Name, "Ubuntu 16.04 x64") {
			conn.cluster.Spec.Cloud.InstanceImage = strconv.Itoa(os.ID)
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
		if key.Name == conn.cluster.Spec.Cloud.SSHKeyName {
			return true, key.ID, nil
		}
	}
	return false, "", nil
}

func (conn *cloudConnector) importPublicKey() error {
	Logger(conn.ctx).Infof("Adding SSH public key")
	resp, err := conn.client.CreateSSHKey(conn.cluster.Spec.Cloud.SSHKeyName, string(SSHKey(conn.ctx).PublicKey))
	if err != nil {
		return errors.Wrap(err, ID(conn.ctx))
	}
	conn.cluster.Status.Cloud.SShKeyExternalID = resp.ID
	Logger(conn.ctx).Infof("New ssh key with name %v and id %v created", conn.cluster.Spec.Cloud.SSHKeyName, resp.ID)
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
	regionID, err := strconv.Atoi(conn.cluster.Spec.Cloud.Zone)
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

func (conn *cloudConnector) getStartupScriptID(ng *api.NodeGroup) (int, error) {
	scriptName := conn.namer.StartupScriptName(ng.Name, ng.Role())

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

func (conn *cloudConnector) createOrUpdateStartupScript(ng *api.NodeGroup, token string) (int, error) {
	scriptName := conn.namer.StartupScriptName(ng.Name, ng.Role())
	script, err := conn.renderStartupScript(ng, token)
	if err != nil {
		return 0, err
	}
	fmt.Println(script)

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

	Logger(conn.ctx).Infof("creating StackScript for NodeGroup %v role %v", ng.Name, ng.Role())
	resp, err := conn.client.CreateStartupScript(scriptName, script, "boot")
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(resp.ID)
}

func (conn *cloudConnector) deleteStartupScript(ng *api.NodeGroup) error {
	scriptName := conn.namer.StartupScriptName(ng.Name, ng.Role())
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

func (conn *cloudConnector) CreateInstance(name, token string, ng *api.NodeGroup) (*api.NodeInfo, error) {
	regionID, err := strconv.Atoi(conn.cluster.Spec.Cloud.Zone)
	if err != nil {
		return nil, err
	}
	planID, err := strconv.Atoi(ng.Spec.Template.Spec.SKU)
	if err != nil {
		return nil, err
	}
	osID, err := strconv.Atoi(conn.cluster.Spec.Cloud.InstanceImage)
	if err != nil {
		return nil, err
	}

	_, sshKeyID, err := conn.getPublicKey()
	if err != nil {
		return nil, err
	}

	scriptID, err := conn.getStartupScriptID(ng)
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
