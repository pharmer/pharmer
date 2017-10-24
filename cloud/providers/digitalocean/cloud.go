package digitalocean

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/appscode/go/errors"
	api "github.com/appscode/pharmer/apis/v1alpha1"
	. "github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/credential"
	"github.com/digitalocean/godo"
	"golang.org/x/crypto/ssh"
	"golang.org/x/oauth2"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

type cloudConnector struct {
	ctx     context.Context
	cluster *api.Cluster
	client  *godo.Client
	namer   namer
}

var _ InstanceManager = &cloudConnector{}

func NewConnector(ctx context.Context, cluster *api.Cluster) (*cloudConnector, error) {
	cred, err := Store(ctx).Credentials().Get(cluster.Spec.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.DigitalOcean{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.New().WithMessagef("Credential %s is invalid. Reason: %v", cluster.Spec.CredentialName, err)
	}
	oauthClient := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: typed.Token(),
	}))
	conn := cloudConnector{
		ctx:     ctx,
		cluster: cluster,
		client:  godo.NewClient(oauthClient),
	}
	if ok, msg := conn.IsUnauthorized(); !ok {
		return nil, fmt.Errorf("credential `%s` does not have necessary autheorization. Reason: %s", cluster.Spec.CredentialName, msg)
	}
	return &conn, nil
}

// Returns true if unauthorized
func (conn *cloudConnector) IsUnauthorized() (bool, string) {
	name := "check-write-access:" + strconv.FormatInt(time.Now().Unix(), 10)
	_, _, err := conn.client.Tags.Create(context.TODO(), &godo.TagCreateRequest{
		Name: name,
	})
	if err != nil {
		return false, "Credential missing WRITE scope"
	}
	conn.client.Tags.Delete(context.TODO(), name)
	return true, ""
}

func (conn *cloudConnector) WaitForInstance(id int, status string) error {
	attempt := 0
	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		attempt++

		droplet, _, err := conn.client.Droplets.Get(context.TODO(), id)
		if err != nil {
			return false, nil
		}
		Logger(conn.ctx).Infof("Attempt %v: Instance `%v` is in status `%s`", attempt, id, droplet.Status)
		if strings.ToLower(droplet.Status) == status {
			return true, nil
		}
		return false, nil
	})
}

func (conn *cloudConnector) getPublicKey() (bool, error) {
	_, resp, err := conn.client.Keys.GetByFingerprint(context.TODO(), SSHKey(conn.ctx).OpensshFingerprint)
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (conn *cloudConnector) importPublicKey() error {
	key, resp, err := conn.client.Keys.Create(context.TODO(), &godo.KeyCreateRequest{
		Name:      conn.cluster.Status.SSHKeyExternalID,
		PublicKey: string(SSHKey(conn.ctx).PublicKey),
	})
	if err != nil {
		return err
	}
	Logger(conn.ctx).Debugln("DO response", resp, " errors", err)
	Logger(conn.ctx).Debugf("Created new ssh key with name=%v and id=%v", conn.cluster.Status.SSHKeyExternalID, key.ID)
	Logger(conn.ctx).Info("SSH public key added")
	return nil
}

func (conn *cloudConnector) getTags() (bool, error) {
	tag := "KubernetesCluster:" + conn.cluster.Name
	_, resp, err := conn.client.Tags.Get(context.TODO(), tag)
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if err != nil {
		// Tag does not already exist
		return false, err
	}
	return true, nil
}

func (conn *cloudConnector) createTags() error {
	tag := "KubernetesCluster:" + conn.cluster.Name
	_, _, err := conn.client.Tags.Create(context.TODO(), &godo.TagCreateRequest{
		Name: tag,
	})
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("Tag %v created", tag)
	return nil
}

func (conn *cloudConnector) getReserveIP(ip string) (bool, error) {
	fip, resp, err := conn.client.FloatingIPs.Get(context.TODO(), ip)
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if err != nil {
		return false, nil
	}
	return fip != nil, nil
}

func (conn *cloudConnector) createReserveIP() (string, error) {
	fip, _, err := conn.client.FloatingIPs.Create(context.TODO(), &godo.FloatingIPCreateRequest{
		Region: conn.cluster.Spec.Cloud.Region,
	})
	if err != nil {
		return "", err
	}
	Logger(conn.ctx).Infof("New floating ip %v reserved", fip.IP)
	return fip.IP, nil
}

func (conn *cloudConnector) CreateInstance(name, token string, ng *api.NodeGroup) (*api.SimpleNode, error) {
	startupScript, err := RenderStartupScript(conn.ctx, conn.cluster, token, ng.Role(), ng.Name, true)
	if err != nil {
		return nil, err
	}
	fmt.Println()
	fmt.Println(startupScript)
	fmt.Println()
	req := &godo.DropletCreateRequest{
		Name:   name,
		Region: conn.cluster.Spec.Cloud.Zone,
		Size:   ng.Spec.Template.Spec.SKU,
		Image:  godo.DropletCreateImage{Slug: conn.cluster.Spec.Cloud.InstanceImage},
		SSHKeys: []godo.DropletCreateSSHKey{
			{Fingerprint: SSHKey(conn.ctx).OpensshFingerprint},
			{Fingerprint: "0d:ff:0d:86:0c:f1:47:1d:85:67:1e:73:c6:0e:46:17"}, // tamal@beast
			{Fingerprint: "c0:19:c1:81:c5:2e:6d:d9:a6:db:3c:f5:c5:fd:c8:1d"}, // tamal@mbp
			{Fingerprint: "f6:66:c5:ad:e6:60:30:d9:ab:2c:7c:75:56:e2:d7:f3"}, // tamal@asus
			{Fingerprint: "80:b6:5a:c8:92:db:aa:fe:5f:d0:2e:99:95:de:ae:ab"}, // sanjid
			{Fingerprint: "93:e6:c6:95:5c:d1:ac:00:5e:23:8c:f7:d2:61:b7:07"}, // dipta
		},
		PrivateNetworking: true,
		IPv6:              false,
		UserData:          startupScript,
	}
	if Env(conn.ctx).IsPublic() {
		req.SSHKeys = []godo.DropletCreateSSHKey{
			{Fingerprint: SSHKey(conn.ctx).OpensshFingerprint},
		}
	}
	droplet, _, err := conn.client.Droplets.Create(context.TODO(), req)
	if err != nil {
		return nil, err
	}
	Logger(conn.ctx).Infof("Droplet %v created", droplet.Name)

	if err = conn.WaitForInstance(droplet.ID, "active"); err != nil {
		return nil, err
	}
	if err = conn.applyTag(droplet.ID); err != nil {
		return nil, err
	}

	// load again to get IP address assigned
	droplet, _, err = conn.client.Droplets.Get(context.TODO(), droplet.ID)
	if err != nil {
		return nil, err
	}
	node := api.SimpleNode{
		Name:       droplet.Name,
		ExternalID: strconv.Itoa(droplet.ID),
	}
	node.PublicIP, err = droplet.PublicIPv4()
	if err != nil {
		return nil, err
	}
	node.PrivateIP, err = droplet.PrivateIPv4()
	if err != nil {
		return nil, err
	}
	return &node, nil
}

func (conn *cloudConnector) applyTag(dropletID int) error {
	_, err := conn.client.Tags.TagResources(context.TODO(), "KubernetesCluster:"+conn.cluster.Name, &godo.TagResourcesRequest{
		Resources: []godo.Resource{
			{
				ID:   strconv.Itoa(dropletID),
				Type: godo.DropletResourceType,
			},
		},
	})
	Logger(conn.ctx).Infof("Tag %v applied to droplet %v", "KubernetesCluster:"+conn.cluster.Name, dropletID)
	return err
}

func (conn *cloudConnector) assignReservedIP(ip string, dropletID int) error {
	action, resp, err := conn.client.FloatingIPActions.Assign(context.TODO(), ip, dropletID)
	if err != nil {
		return errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	Logger(conn.ctx).Debugln("do response", resp, " errors", err)
	Logger(conn.ctx).Debug("Created droplet with name", action.String())
	Logger(conn.ctx).Infof("Reserved ip %v assigned to droplet %v", ip, dropletID)
	return nil
}

// reboot does not seem to run /etc/rc.local
func (conn *cloudConnector) reboot(id int) error {
	Logger(conn.ctx).Infof("Rebooting instance %v", id)
	action, _, err := conn.client.DropletActions.Reboot(context.TODO(), id)
	if err != nil {
		return errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	Logger(conn.ctx).Debugf("Instance status %v, %v", action, err)
	Logger(conn.ctx).Infof("Instance %v reboot status %v", action.ResourceID, action.Status)
	return nil
}

func (conn *cloudConnector) releaseReservedIP(ip string) error {
	resp, err := conn.client.FloatingIPs.Delete(context.TODO(), ip)
	Logger(conn.ctx).Debugln("DO response", resp, " errors", err)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	Logger(conn.ctx).Infof("Floating ip %v deleted", ip)
	return nil
}

func (conn *cloudConnector) deleteSSHKey() error {
	err := wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		_, err := conn.client.Keys.DeleteByFingerprint(context.TODO(), SSHKey(conn.ctx).OpensshFingerprint)
		return err == nil, nil
	})
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("SSH key for cluster %v deleted", conn.cluster.Name)
	return nil
}

func (conn *cloudConnector) DeleteInstanceByProviderID(providerID string) error {
	if providerID == "" {
		return errors.New("providerID cannot be empty string")
	}

	parts := strings.SplitN(providerID, "://", 2)
	if len(parts) != 2 {
		return fmt.Errorf("skipping deleting node with providerID `%s`", providerID)
	}
	dropletID, err := strconv.Atoi(parts[1]) // TODO: FixIt!
	if err != nil {
		return err
	}
	_, err = conn.client.Droplets.Delete(context.TODO(), dropletID)
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("Droplet %v deleted", dropletID)
	return nil
}

func (conn *cloudConnector) ExecuteSSHCommand(command string, instance *core.Node) (string, error) {
	var ip string = ""
	for _, addr := range instance.Status.Addresses {
		if addr.Type == core.NodeExternalIP {
			ip = addr.Address
			break
		}
	}
	if ip == "" {
		return "", fmt.Errorf("no IP found for ssh")
	}
	keySigner, _ := ssh.ParsePrivateKey(SSHKey(conn.ctx).PrivateKey)
	config := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(keySigner),
		},
	}
	return ExecuteTCPCommand(command, fmt.Sprintf("%v:%v", ip, 22), config)
}
