package digitalocean

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	. "github.com/appscode/go/context"
	"github.com/digitalocean/godo"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/credential"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

var errLBNotFound = errors.New("loadbalancer not found")

type cloudConnector struct {
	ctx     context.Context
	cluster *api.Cluster
	client  *godo.Client
}

var _ InstanceManager = &cloudConnector{}

func NewConnector(ctx context.Context, cluster *api.Cluster, owner string) (*cloudConnector, error) {
	cred, err := Store(ctx).Owner(owner).Credentials().Get(cluster.ClusterConfig().CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.DigitalOcean{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.Wrapf(err, "credential %s is invalid", cluster.ClusterConfig().CredentialName)
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
		return nil, errors.Errorf("credential `%s` does not have necessary autheorization. Reason: %s", cluster.ClusterConfig().CredentialName, msg)
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

func (conn *cloudConnector) CreateCredentialSecret(kc kubernetes.Interface, data map[string]string) error {
	return nil
}

func PrepareCloud(ctx context.Context, clusterName, owner string) (*cloudConnector, error) {
	var err error
	var conn *cloudConnector
	cluster, err := Store(ctx).Owner(owner).Clusters().Get(clusterName)
	if err != nil {
		return conn, fmt.Errorf("cluster `%s` does not exist. Reason: %v", clusterName, err)
	}
	//cm.cluster = cluster

	if ctx, err = LoadCACertificates(ctx, cluster, owner); err != nil {
		return conn, err
	}
	/*if cm.ctx, err = LoadEtcdCertificate(cm.ctx, cm.cluster); err != nil {
		return err
	}*/
	if ctx, err = LoadSSHKey(ctx, cluster, owner); err != nil {
		return conn, err
	}
	/*if cm.ctx, err = LoadSaKey(cm.ctx, cm.cluster); err != nil {
		return err
	}*/

	if conn, err = NewConnector(ctx, cluster, owner); err != nil {
		return nil, err
	}
	//cm.namer = namer{cluster: cm.cluster}
	return conn, nil
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

// ---------------------------------------------------------------------------------------------------------------------

func (conn *cloudConnector) getPublicKey() (bool, int, error) {
	key, resp, err := conn.client.Keys.GetByFingerprint(context.TODO(), SSHKey(conn.ctx).OpensshFingerprint)
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return false, 0, nil
	}
	if err != nil {
		return false, 0, err
	}
	return true, key.ID, nil
}

func (conn *cloudConnector) importPublicKey() (string, error) {
	Logger(conn.ctx).Infof("Adding SSH public key")
	id, _, err := conn.client.Keys.Create(context.TODO(), &godo.KeyCreateRequest{
		//	Name:      conn.cluster.Spec.Cloud.SSHKeyName,
		PublicKey: string(SSHKey(conn.ctx).PublicKey),
	})
	if err != nil {
		return "", err
	}
	Logger(conn.ctx).Info("SSH public key added")
	return strconv.Itoa(id.ID), nil
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

// ---------------------------------------------------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------------------------------------------------

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
		Region: conn.cluster.ClusterConfig().Cloud.Region,
	})
	if err != nil {
		return "", err
	}
	Logger(conn.ctx).Infof("New floating ip %v reserved", fip.IP)
	return fip.IP, nil
}

func (conn *cloudConnector) assignReservedIP(ip string, dropletID int) error {
	_, _, err := conn.client.FloatingIPActions.Assign(context.TODO(), ip, dropletID)
	if err != nil {
		return errors.Wrap(err, ID(conn.ctx))
	}
	Logger(conn.ctx).Infof("Reserved ip %v assigned to droplet %v", ip, dropletID)
	return nil
}

func (conn *cloudConnector) releaseReservedIP(ip string) error {
	resp, err := conn.client.FloatingIPs.Delete(context.TODO(), ip)
	Logger(conn.ctx).Debugln("DO response", resp, " errors", err)
	if err != nil {
		return errors.WithStack(err)
	}
	Logger(conn.ctx).Infof("Floating ip %v deleted", ip)
	return nil
}

// ---------------------------------------------------------------------------------------------------------------------

func (conn *cloudConnector) CreateInstance(cluster *api.Cluster, machine *clusterv1.Machine, token string) (*api.NodeInfo, error) {
	script, err := conn.renderStartupScript(cluster, machine, token)
	if err != nil {
		return nil, err
	}

	fmt.Println()
	fmt.Println(script)
	fmt.Println()

	machineConfig, err := machineProviderFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		return nil, err
	}
	req := &godo.DropletCreateRequest{
		Name:   machine.Name,
		Region: machineConfig.Region,
		Size:   machineConfig.Size,
		Image:  godo.DropletCreateImage{Slug: machineConfig.Image},
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
		UserData:          script,
	}
	if Env(conn.ctx).IsPublic() {
		req.SSHKeys = []godo.DropletCreateSSHKey{
			{Fingerprint: SSHKey(conn.ctx).OpensshFingerprint},
		}
	}
	host, _, err := conn.client.Droplets.Create(context.TODO(), req)
	if err != nil {
		return nil, err
	}
	Logger(conn.ctx).Infof("Droplet %v created", host.Name)

	if err = conn.WaitForInstance(host.ID, "active"); err != nil {
		return nil, err
	}
	if err = conn.applyTag(host.ID); err != nil {
		return nil, err
	}

	// load again to get IP address assigned
	host, _, err = conn.client.Droplets.Get(context.TODO(), host.ID)
	if err != nil {
		return nil, err
	}
	node := api.NodeInfo{
		Name:       host.Name,
		ExternalID: strconv.Itoa(host.ID),
	}
	node.PublicIP, err = host.PublicIPv4()
	if err != nil {
		return nil, err
	}
	node.PrivateIP, err = host.PrivateIPv4()
	if err != nil {
		return nil, err
	}
	return &node, nil
}

func (conn *cloudConnector) instanceIfExists(machine *clusterv1.Machine) (*godo.Droplet, error) {
	//identifyingMachine := machine

	droplets, _, err := conn.client.Droplets.List(oauth2.NoContext, &godo.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, droplet := range droplets {
		if droplet.Name == machine.Name {
			d, _, err := conn.client.Droplets.Get(oauth2.NoContext, droplet.ID)
			if err != nil {
				return nil, err
			}
			return d, nil
		}
	}

	return nil, fmt.Errorf("no droplet found with %v name", machine.Name)
}

func (conn *cloudConnector) DeleteInstanceByProviderID(providerID string) error {
	dropletID, err := dropletIDFromProviderID(providerID)
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

// dropletIDFromProviderID returns a droplet's ID from providerID.
//
// The providerID spec should be retrievable from the Kubernetes
// node object. The expected format is: digitalocean://droplet-id
// ref: https://github.com/digitalocean/digitalocean-cloud-controller-manager/blob/f9a9856e99c9d382db3777d678f29d85dea25e91/do/droplets.go#L211
func dropletIDFromProviderID(providerID string) (int, error) {
	if providerID == "" {
		return 0, errors.New("providerID cannot be empty string")
	}

	split := strings.Split(providerID, "/")
	if len(split) != 3 {
		return 0, errors.Errorf("unexpected providerID format: %s, format should be: digitalocean://12345", providerID)
	}

	// since split[0] is actually "digitalocean:"
	if strings.TrimSuffix(split[0], ":") != UID {
		return 0, errors.Errorf("provider name from providerID should be digitalocean: %s", providerID)
	}

	return strconv.Atoi(split[2])
}

func (conn *cloudConnector) deleteInstance(ctx context.Context, id int) error {
	_, err := conn.client.Droplets.Delete(ctx, id)
	return err
}

// ---------------------------------------------------------------------------------------------------------------------

func (conn *cloudConnector) createLoadBalancer(ctx context.Context, name string) (string, error) {
	lb, err := conn.lbByName(ctx, name)
	if err != nil {
		if err == errLBNotFound {
			lbRequest, err := conn.buildLoadBalancerRequest(name)
			if err != nil {
				return "", err
			}
			lb, _, err := conn.client.LoadBalancers.Create(ctx, lbRequest)
			if err != nil {
				return "", err
			}
			if lb, err = conn.waitActive(lb.ID); err != nil {
				return "", err
			}
			return lb.IP, nil
		}
	}

	if lb.Status != "active" {
		if lb, err = conn.waitActive(lb.ID); err != nil {
			return "", err
		}
	}
	return lb.IP, nil
}

func (conn *cloudConnector) deleteLoadBalancer(ctx context.Context, name string) error {
	lb, err := conn.lbByName(ctx, name)
	if err != nil {
		return err
	}
	_, err = conn.client.LoadBalancers.Delete(ctx, lb.ID)
	return err
}

func (conn *cloudConnector) addNodeToBalancer(ctx context.Context, lbName string, id int) error {
	lb, err := conn.lbByName(ctx, lbName)
	if err != nil {
		return err
	}
	lb.DropletIDs = append(lb.DropletIDs, id)
	_, err = conn.client.LoadBalancers.AddDroplets(ctx, lb.ID, id)
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("Added master %v to loadbalancer %v", id, lbName)

	return nil
}

func (conn *cloudConnector) lbByName(ctx context.Context, name string) (*godo.LoadBalancer, error) {
	lbs, _, err := conn.client.LoadBalancers.List(ctx, &godo.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, lb := range lbs {
		if lb.Name == name {
			return &lb, nil
		}
	}

	return nil, errLBNotFound
}

// buildLoadBalancerRequest returns a *godo.LoadBalancerRequest to balance
// requests for service across nodes.
func (conn *cloudConnector) buildLoadBalancerRequest(lbName string) (*godo.LoadBalancerRequest, error) {

	forwardingRules := []godo.ForwardingRule{
		{
			EntryProtocol:  "tcp",
			EntryPort:      kubeadmapi.DefaultAPIBindPort,
			TargetProtocol: "tcp",
			TargetPort:     kubeadmapi.DefaultAPIBindPort,
			//CertificateID  string `json:"certificate_id,omitempty"`
			TlsPassthrough: false,
		},
	}

	healthCheck := &godo.HealthCheck{
		Protocol:               "tcp",
		Port:                   kubeadmapi.DefaultAPIBindPort,
		CheckIntervalSeconds:   3,
		ResponseTimeoutSeconds: 5,
		HealthyThreshold:       5,
		UnhealthyThreshold:     3,
	}

	stickySessions := &godo.StickySessions{
		Type: "none",
		//CookieName:       name,
		//CookieTtlSeconds: ttl,
	}

	algorithm := "least_connections"
	//algorithm := "round_robin"

	//	redirectHttpToHttps := getRedirectHttpToHttps(service)
	clusterConfig := conn.cluster.ClusterConfig()

	return &godo.LoadBalancerRequest{
		Name:                lbName,
		DropletIDs:          []int{},
		Region:              clusterConfig.Cloud.Region,
		ForwardingRules:     forwardingRules,
		HealthCheck:         healthCheck,
		StickySessions:      stickySessions,
		Algorithm:           algorithm,
		RedirectHttpToHttps: false, //redirectHttpToHttps,
	}, nil
}

func (conn *cloudConnector) waitActive(lbID string) (*godo.LoadBalancer, error) {
	attempt := 0
	err := wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		attempt++

		lb, _, err := conn.client.LoadBalancers.Get(context.TODO(), lbID)
		if err != nil {
			return false, nil
		}
		Logger(conn.ctx).Infof("Attempt %v: LoadBalancer `%v` is in status `%s`", attempt, lbID, lb.Status)
		fmt.Println(lb.String())
		if strings.ToLower(lb.Status) == "active" {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return nil, err
	}
	lb, _, err := conn.client.LoadBalancers.Get(context.TODO(), lbID)
	return lb, err

}
