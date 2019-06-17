package digitalocean

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/appscode/go/log"
	"github.com/digitalocean/godo"
	"github.com/pharmer/cloud/pkg/credential"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	doCapi "github.com/pharmer/pharmer/apis/v1beta1/digitalocean"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/util"
)

var errLBNotFound = errors.New("loadbalancer not found")

type cloudConnector struct {
	*cloud.CloudManager
	client *godo.Client
	namer  namer
}

func newConnector(cm *ClusterManager) (*cloudConnector, error) {
	cluster := cm.Cluster

	cred, err := store.StoreProvider.Credentials().Get(cluster.ClusterConfig().CredentialName)
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

	namer := namer{cluster: cluster}

	conn := cloudConnector{
		CloudManager: cm.CloudManager,
		namer:        namer,
		client:       godo.NewClient(oauthClient),
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
	if _, err := conn.client.Tags.Delete(context.TODO(), name); err != nil {
		return false, err.Error()
	}
	return true, ""
}

func (conn *cloudConnector) CreateCredentialSecret(kc kubernetes.Interface, data map[string]string) error {
	return nil
}

func (conn *cloudConnector) WaitForInstance(id int, status string) error {
	attempt := 0
	return wait.PollImmediate(api.RetryInterval, api.RetryTimeout, func() (bool, error) {
		attempt++

		droplet, _, err := conn.client.Droplets.Get(context.TODO(), id)
		if err != nil {
			return false, nil
		}
		log.Infof("Attempt %v: Instance `%v` is in status `%s`", attempt, id, droplet.Status)
		if strings.ToLower(droplet.Status) == status {
			return true, nil
		}
		return false, nil
	})
}

// ---------------------------------------------------------------------------------------------------------------------

func (conn *cloudConnector) getPublicKey() (bool, int, error) {
	key, resp, err := conn.client.Keys.GetByFingerprint(context.TODO(), conn.Certs.SSHKey.OpensshFingerprint)
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return false, 0, nil
	}
	if err != nil {
		return false, 0, err
	}
	return true, key.ID, nil
}

func (conn *cloudConnector) importPublicKey() (string, error) {
	log.Infof("Adding SSH public key")
	id, _, err := conn.client.Keys.Create(context.TODO(), &godo.KeyCreateRequest{
		//	Name:      conn.Cluster.Spec.Cloud.SSHKeyName,
		PublicKey: string(conn.Certs.SSHKey.PublicKey),
	})
	if err != nil {
		return "", err
	}
	log.Info("SSH public key added")
	return strconv.Itoa(id.ID), nil
}

func (conn *cloudConnector) deleteSSHKey() error {
	err := wait.PollImmediate(api.RetryInterval, api.RetryTimeout, func() (bool, error) {
		_, err := conn.client.Keys.DeleteByFingerprint(context.TODO(), conn.Certs.SSHKey.OpensshFingerprint)
		return err == nil, nil
	})
	if err != nil {
		return err
	}
	log.Infof("SSH key for cluster %v deleted", conn.Cluster.Name)
	return nil
}

// ---------------------------------------------------------------------------------------------------------------------

func (conn *cloudConnector) getTags() (bool, error) {
	tag := "KubernetesCluster:" + conn.Cluster.Name
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
	tag := "KubernetesCluster:" + conn.Cluster.Name
	_, _, err := conn.client.Tags.Create(context.TODO(), &godo.TagCreateRequest{
		Name: tag,
	})
	if err != nil {
		return err
	}
	log.Infof("Tag %v created", tag)
	return nil
}

func (conn *cloudConnector) CreateInstance(cluster *api.Cluster, machine *clusterv1.Machine, script string) (*api.NodeInfo, error) {
	machineConfig, err := doCapi.MachineConfigFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		return nil, err
	}
	req := &godo.DropletCreateRequest{
		Name:   machine.Name,
		Region: machineConfig.Region,
		Size:   machineConfig.Size,
		Image:  godo.DropletCreateImage{Slug: machineConfig.Image},
		SSHKeys: []godo.DropletCreateSSHKey{
			{Fingerprint: conn.Certs.SSHKey.OpensshFingerprint},
		},
		PrivateNetworking: true,
		IPv6:              false,
		UserData:          script,
		Tags: []string{
			"KubernetesCluster:" + cluster.Name,
		},
	}

	if util.IsControlPlaneMachine(machine) {
		req.Tags = append(req.Tags, cluster.Name+"-master")
	}

	host, _, err := conn.client.Droplets.Create(context.TODO(), req)
	if err != nil {
		return nil, err
	}
	log.Infof("Droplet %v created", host.Name)

	if err = conn.WaitForInstance(host.ID, "active"); err != nil {
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

	droplets, _, err := conn.client.Droplets.List(context.Background(), &godo.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, droplet := range droplets {
		if droplet.Name == machine.Name {
			d, _, err := conn.client.Droplets.Get(context.Background(), droplet.ID)
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
	log.Infof("Droplet %v deleted", dropletID)
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

func (conn *cloudConnector) createLoadBalancer(ctx context.Context, name string) (*godo.LoadBalancer, error) {
	lb, err := conn.lbByName(ctx, name)
	if err != nil {
		if err == errLBNotFound {
			lbRequest := conn.buildLoadBalancerRequest(name)
			lb, _, err := conn.client.LoadBalancers.Create(ctx, lbRequest)
			if err != nil {
				return nil, err
			}
			if lb, err = conn.waitActive(lb.ID); err != nil {
				return nil, err
			}
			return lb, nil
		}
	}

	if lb.Status != "active" {
		if lb, err = conn.waitActive(lb.ID); err != nil {
			return nil, err
		}
	}
	return lb, nil
}

func (conn *cloudConnector) deleteLoadBalancer(ctx context.Context, name string) error {
	lb, err := conn.lbByName(ctx, name)
	if err != nil {
		return err
	}
	_, err = conn.client.LoadBalancers.Delete(ctx, lb.ID)
	return err
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
func (conn *cloudConnector) buildLoadBalancerRequest(lbName string) *godo.LoadBalancerRequest {

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
	clusterConfig := conn.Cluster.ClusterConfig()

	return &godo.LoadBalancerRequest{
		Name:                lbName,
		DropletIDs:          []int{},
		Region:              clusterConfig.Cloud.Region,
		ForwardingRules:     forwardingRules,
		HealthCheck:         healthCheck,
		StickySessions:      stickySessions,
		Algorithm:           algorithm,
		RedirectHttpToHttps: false, //redirectHttpToHttps,
		Tag:                 conn.Cluster.Name + "-master",
		Tags:                []string{conn.Cluster.Name + "-master"},
	}
}

func (conn *cloudConnector) loadBalancerUpdated(lb *godo.LoadBalancer) bool {
	defaultSpecs := conn.buildLoadBalancerRequest(conn.namer.LoadBalancerName())

	if lb.Algorithm != defaultSpecs.Algorithm {
		return true
	}
	if lb.Region.Slug != defaultSpecs.Region {
		return true
	}
	if !reflect.DeepEqual(lb.ForwardingRules, defaultSpecs.ForwardingRules) {
		return true
	}
	if !reflect.DeepEqual(lb.HealthCheck, defaultSpecs.HealthCheck) {
		return true
	}
	if !reflect.DeepEqual(lb.StickySessions, defaultSpecs.StickySessions) {
		return true
	}
	if lb.RedirectHttpToHttps != defaultSpecs.RedirectHttpToHttps {
		return true
	}

	return false
}

func (conn *cloudConnector) waitActive(lbID string) (*godo.LoadBalancer, error) {
	attempt := 0
	err := wait.PollImmediate(api.RetryInterval, api.RetryTimeout, func() (bool, error) {
		attempt++

		lb, _, err := conn.client.LoadBalancers.Get(context.TODO(), lbID)
		if err != nil {
			return false, nil
		}
		log.Infof("Attempt %v: LoadBalancer `%v` is in status `%s`", attempt, lbID, lb.Status)
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
