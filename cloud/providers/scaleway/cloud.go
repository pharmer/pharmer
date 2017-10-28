package scaleway

import (
	"context"
	"fmt"
	"strings"

	sshtools "github.com/appscode/go/crypto/ssh"
	"github.com/appscode/go/errors"
	. "github.com/appscode/go/types"
	api "github.com/appscode/pharmer/apis/v1alpha1"
	. "github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/credential"
	sapi "github.com/scaleway/scaleway-cli/pkg/api"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

type cloudConnector struct {
	ctx          context.Context
	cluster      *api.Cluster
	client       *sapi.ScalewayAPI
	bootscriptID string
}

func NewConnector(ctx context.Context, cluster *api.Cluster) (*cloudConnector, error) {
	cred, err := Store(ctx).Credentials().Get(cluster.Spec.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.Scaleway{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.New().WithMessagef("Credential %s is invalid. Reason: %v", cluster.Spec.CredentialName, err)
	}

	client, err := sapi.NewScalewayAPI(typed.Organization(), typed.Token(), "pharmer", cluster.Spec.Cloud.Zone)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(ctx).Err()
	}
	return &cloudConnector{
		ctx:    ctx,
		client: client,
	}, nil
}

func (conn *cloudConnector) getInstanceImage() (string, error) {
	imgs, err := conn.client.GetMarketPlaceImages("")
	if err != nil {
		return "", errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	for _, img := range imgs.Images {
		if img.Name == "Ubuntu Xenial" {
			for _, v := range img.Versions {
				for _, li := range v.LocalImages {
					if li.Arch == "x86_64" && li.Zone == conn.cluster.Spec.Cloud.Zone {
						return li.ID, nil
					}
				}
			}
		}
	}
	return "", errors.New("Debian Jessie not found for Scaleway").WithContext(conn.ctx).Err()
}

// http://devhub.scaleway.com/#/bootscripts
func (conn *cloudConnector) DetectBootscript() error {
	scripts, err := conn.client.GetBootscripts()
	if err != nil {
		return errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	for _, s := range *scripts {
		// x86_64 4.8.3 docker #1
		if s.Arch == "x86_64" && strings.Contains(s.Title, "docker") {
			conn.bootscriptID = s.Identifier
			return nil
		}
	}
	return errors.New("Docker bootscript not found for Scaleway").WithContext(conn.ctx).Err()
}

func (conn *cloudConnector) waitForInstance(id, status string) error {
	attempt := 0
	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		attempt++

		server, err := conn.client.GetServer(id)
		if err != nil {
			return false, nil
		}
		Logger(conn.ctx).Infof("Attempt %v: Instance `%v` is in status `%s`", attempt, id, server.State)
		if strings.ToLower(server.State) == status {
			return true, nil
		}
		return false, nil
	})
}

// ---------------------------------------------------------------------------------------------------------------------

func (conn *cloudConnector) getPublicKey() (bool, string, error) {
	user, err := conn.client.GetUser()
	if err != nil {
		return false, "", err
	}
	for _, k := range user.SSHPublicKeys {
		if k.Fingerprint == SSHKey(conn.ctx).OpensshFingerprint {
			return true, k.Key, nil
		}
	}
	return false, "", nil
}

func (conn *cloudConnector) importPublicKey() error {
	Logger(conn.ctx).Infof("Adding SSH public key")
	err := wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		user, err := conn.client.GetUser()
		if err != nil {
			return false, nil // retry
		}
		sshPubKeys := make([]sapi.ScalewayKeyDefinition, len(user.SSHPublicKeys)+1)
		for i, k := range user.SSHPublicKeys {
			sshPubKeys[i] = sapi.ScalewayKeyDefinition{Key: k.Key}
		}
		sshPubKeys[len(user.SSHPublicKeys)] = sapi.ScalewayKeyDefinition{
			Key: string(SSHKey(conn.ctx).PublicKey),
		}
		err = conn.client.PatchUserSSHKey(user.ID, sapi.ScalewayUserPatchSSHKeyDefinition{
			SSHPublicKeys: sshPubKeys,
		})
		return err == nil, nil
	})
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("New ssh key with fingerprint %v created", SSHKey(conn.ctx).OpensshFingerprint)
	return nil
}

func (conn *cloudConnector) deleteSSHKey(key string) error {
	Logger(conn.ctx).Infof("Deleting SSH key for cluster", conn.cluster.Name)
	err := wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		user, err := conn.client.GetUser()
		if err != nil {
			return false, nil // retry
		}
		sshPubKeys := make([]sapi.ScalewayKeyDefinition, 0, len(user.SSHPublicKeys))
		for _, k := range user.SSHPublicKeys {
			if k.Key != key {
				sshPubKeys = append(sshPubKeys, sapi.ScalewayKeyDefinition{Key: k.Key})
			}
		}
		err = conn.client.PatchUserSSHKey(user.ID, sapi.ScalewayUserPatchSSHKeyDefinition{
			SSHPublicKeys: sshPubKeys,
		})
		return err == nil, nil
	})
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("SSH key for cluster %v deleted", conn.cluster.Name)
	return nil
}

// ---------------------------------------------------------------------------------------------------------------------

func (conn *cloudConnector) createReserveIP() (string, error) {
	Logger(conn.ctx).Infof("Reserving Floating IP")
	fip, err := conn.client.NewIP()
	if err != nil {
		return "", errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	Logger(conn.ctx).Infof("New floating ip %v reserved", fip.IP)
	return fip.IP.ID, nil
}

func (conn *cloudConnector) releaseReservedIP(ip string) error {
	ips, err := conn.client.GetIPS()
	if err != nil {
		return errors.FromErr(err).Err()
	}
	for _, i := range ips.IPS {
		if i.Address == ip && i.Server == nil {
			err = conn.client.DeleteIP(ip)
			if err != nil {
				return errors.FromErr(err).Err()
			}
		}
	}
	Logger(conn.ctx).Infof("Floating ip %v deleted", ip)
	return nil
}

// ---------------------------------------------------------------------------------------------------------------------

func (conn *cloudConnector) storeStartupScript(ng *api.NodeGroup, serverID, token string) error {
	Logger(conn.ctx).Infof("Storing startup script for server %v", serverID)
	script, err := conn.renderStartupScript(ng, token)
	if err != nil {
		return err
	}
	key := "kubernetes_startupscript.sh"
	return conn.client.PatchUserdata(serverID, key, []byte(script), false)
}

//func (conn *cloudConnector) executeStartupScript(instance *api.Node, signer ssh.Signer) error {
//	Logger(conn.ctx).Infof("SSH executing start command %v", instance.Status.PublicIP+":22")
//
//	stdOut, stdErr, code, err := sshtools.Exec(`/usr/bin/curl 169.254.42.42/user_data/kubernetes_startupscript.sh --local-port 1-1024 2> /dev/null | bash`, "root", instance.Status.PublicIP+":22", signer)
//	Logger(conn.ctx).Infoln(stdOut, stdErr, code)
//	if err != nil {
//		return errors.FromErr(err).WithContext(conn.ctx).Err()
//	}
//	return nil
//}

// ---------------------------------------------------------------------------------------------------------------------

// func (conn *cloudConnector) createInstance(name, role, sku string, ipid ...string) (string, error) {
func (conn *cloudConnector) CreateInstance(name, token string, ng *api.NodeGroup) (*api.NodeInfo, error) {
	publicIPID := ""
	if ng.Role() == api.RoleMaster && ng.Spec.Template.Spec.ExternalIPType == api.IPTypeReserved {
		if len(conn.cluster.Status.ReservedIPs) == 0 {
			reservedIP, err := conn.createReserveIP()
			if err != nil {
				return nil, err
			}
			publicIPID = reservedIP
			conn.cluster.Status.APIAddresses = append(conn.cluster.Status.APIAddresses, core.NodeAddress{
				Type:    core.NodeExternalIP,
				Address: reservedIP,
			})
			conn.cluster.Status.ReservedIPs = append(conn.cluster.Status.ReservedIPs, api.ReservedIP{
				IP: reservedIP,
			})
		}
	}

	req := sapi.ScalewayServerDefinition{
		Name:              name,
		Image:             StringP(conn.cluster.Spec.Cloud.InstanceImage),
		DynamicIPRequired: TrueP(),
		Bootscript:        StringP(conn.bootscriptID),
		Tags:              []string{"KubernetesCluster:" + conn.cluster.Name},
		CommercialType:    ng.Spec.Template.Spec.SKU,
		PublicIP:          publicIPID,
		// Organization:   organization,
		//Volumes map[string]string `json:"volumes,omitempty"`
		//EnableIPV6 bool `json:"enable_ipv6,omitempty"`
		//SecurityGroup string `json:"security_group,omitempty"`
	}
	serverID, err := conn.client.PostServer(req)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	err = conn.storeStartupScript(ng, serverID, token)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	err = conn.client.PostServerAction(serverID, "poweron")
	if err != nil {
		return nil, errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	Logger(conn.ctx).Infof("Instance %v created", name)

	err = conn.waitForInstance(serverID, "running")
	if err != nil {
		return nil, errors.FromErr(err).WithContext(conn.ctx).Err()
	}

	host, err := conn.client.GetServer(serverID)
	if err != nil {
		return nil, err
	}

	signer, err := sshtools.MakePrivateKeySignerFromBytes(SSHKey(conn.ctx).PrivateKey)
	if err != nil {
		return nil, err
	}

	Logger(conn.ctx).Infof("SSH executing start command %v", host.PublicAddress.IP+":22")
	stdOut, stdErr, code, err := sshtools.Exec(`/usr/bin/curl 169.254.42.42/user_data/kubernetes_startupscript.sh --local-port 1-1024 2> /dev/null | bash`, "root", host.PublicAddress.IP+":22", signer)
	if err != nil {
		return nil, err
	}
	Logger(conn.ctx).Debugln(stdOut, stdErr, code)

	node := api.NodeInfo{
		Name:       host.Name,
		ExternalID: host.Identifier,
		PublicIP:   host.PublicAddress.IP,
		PrivateIP:  host.PrivateIP,
	}
	return &node, nil
}

func (conn *cloudConnector) DeleteInstanceByProviderID(providerID string) error {
	dropletID, err := serverIDFromProviderID(providerID)
	if err != nil {
		return err
	}
	err = conn.client.DeleteServerForce(dropletID)
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
