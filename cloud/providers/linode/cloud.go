package linode

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/appscode/go/types"
	"github.com/linode/linodego"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"k8s.io/apimachinery/pkg/util/wait"
	"pharmer.dev/cloud/pkg/credential"
	api "pharmer.dev/pharmer/apis/v1alpha1"
	linode_config "pharmer.dev/pharmer/apis/v1alpha1/linode"
	"pharmer.dev/pharmer/cloud"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

var errLBNotFound = errors.New("loadbalancer not found")

type cloudConnector struct {
	*cloud.Scope
	namer  namer
	client *linodego.Client
}

func newconnector(cm *ClusterManager) (*cloudConnector, error) {
	log := cm.Logger
	cluster := cm.Cluster

	cred, err := cm.GetCredential()
	if err != nil {
		log.Error(err, "failed to get credential from store")
		return nil, err
	}
	typed := credential.Linode{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		log.Error(err, "credential is invalid", "credential", cluster.ClusterConfig())
		return nil, errors.Errorf("credential %s is invalid. Reason: %v", cluster.ClusterConfig().CredentialName, err)
	}

	namer := namer{cluster: cluster}

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: typed.APIToken()})

	oauth2Client := &http.Client{
		Transport: &oauth2.Transport{
			Source: tokenSource,
		},
	}

	c := linodego.NewClient(oauth2Client)

	return &cloudConnector{
		Scope:  cm.Scope,
		namer:  namer,
		client: &c,
	}, nil
}

func (conn *cloudConnector) waitForStatus(id int, status linodego.InstanceStatus) error {
	log := conn.Logger
	attempt := 0
	log.Info("waiting for instance status", "status", status)
	return wait.PollImmediate(api.RetryInterval, api.RetryTimeout, func() (bool, error) {
		attempt++

		instance, err := conn.client.GetInstance(context.Background(), id)
		if err != nil {
			return false, nil
		}
		if instance == nil {
			return false, nil
		}
		conn.Logger.V(4).Info("current instance state", "instance", instance.Label, "status", instance.Status, "attempt", attempt)
		if instance.Status == status {
			log.Info("current instance status", "status", status)
			return true, nil
		}
		return false, nil
	})
}

func (conn *cloudConnector) getStartupScriptID(machine *clusterv1.Machine) (int, error) {
	machineConfig, err := linode_config.MachineConfigFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		return 0, err
	}
	scriptName := conn.namer.StartupScriptName(machine.Name, string(machineConfig.Roles[0]))
	filter := fmt.Sprintf(`{"label" : "%v"}`, scriptName)
	listOpts := &linodego.ListOptions{PageOptions: nil, Filter: filter}

	scripts, err := conn.client.ListStackscripts(context.Background(), listOpts)
	if err != nil {
		return 0, err
	}

	if len(scripts) > 1 {
		return 0, errors.Errorf("multiple stackscript found with label %v", scriptName)
	} else if len(scripts) == 0 {
		return 0, errors.Errorf("no stackscript found with label %v", scriptName)
	}
	return scripts[0].ID, nil
}

func (conn *cloudConnector) createOrUpdateStackScript(machine *clusterv1.Machine, script string) (int, error) {
	machineConfig, err := linode_config.MachineConfigFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		return 0, err
	}
	scriptName := conn.namer.StartupScriptName(machine.Name, string(machineConfig.Roles[0]))
	filter := fmt.Sprintf(`{"label" : "%v"}`, scriptName)
	listOpts := &linodego.ListOptions{PageOptions: nil, Filter: filter}

	scripts, err := conn.client.ListStackscripts(context.Background(), listOpts)
	if err != nil {
		return 0, err
	}

	if len(scripts) > 1 {
		return 0, errors.Errorf("multiple stackscript found with label %v", scriptName)
	} else if len(scripts) == 0 {
		createOpts := linodego.StackscriptCreateOptions{
			Label:       scriptName,
			Description: fmt.Sprintf("Startup script for NodeGroup %s of Cluster %s", machine.Name, conn.Cluster.Name),
			Images:      []string{conn.Cluster.ClusterConfig().Cloud.InstanceImage},
			Script:      script,
		}
		stackScript, err := conn.client.CreateStackscript(context.Background(), createOpts)
		if err != nil {
			return 0, err
		}
		conn.Logger.Info("Stack script created", "role", string(machineConfig.Roles[0]))
		return stackScript.ID, nil
	}

	updateOpts := scripts[0].GetUpdateOptions()
	updateOpts.Script = script

	stackScript, err := conn.client.UpdateStackscript(context.Background(), scripts[0].ID, updateOpts)
	if err != nil {
		return 0, err
	}

	conn.Logger.Info("Stack script updated", "role", string(machineConfig.Roles[0]))
	return stackScript.ID, nil
}

func (conn *cloudConnector) DeleteStackScript(machineName string, role string) error {
	scriptName := conn.namer.StartupScriptName(machineName, role)
	filter := fmt.Sprintf(`{"label" : "%v"}`, scriptName)
	listOpts := &linodego.ListOptions{PageOptions: nil, Filter: filter}

	scripts, err := conn.client.ListStackscripts(context.Background(), listOpts)
	if err != nil {
		return err
	}
	for _, script := range scripts {
		if err := conn.client.DeleteStackscript(context.Background(), script.ID); err != nil {
			return err
		}
	}
	return nil
}

func (conn *cloudConnector) CreateInstance(machine *clusterv1.Machine, script string) (*api.NodeInfo, error) {
	if _, err := conn.createOrUpdateStackScript(machine, script); err != nil {
		return nil, err
	}
	scriptID, err := conn.getStartupScriptID(machine)
	if err != nil {
		return nil, err
	}
	machineConfig, err := linode_config.MachineConfigFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		return nil, err
	}
	createOpts := linodego.InstanceCreateOptions{
		Label:    machine.Name,
		Region:   conn.Cluster.ClusterConfig().Cloud.Zone,
		Type:     machineConfig.Type,
		RootPass: conn.Cluster.ClusterConfig().Cloud.Linode.RootPassword,
		AuthorizedKeys: []string{
			string(conn.Certs.SSHKey.PublicKey),
		},
		StackScriptData: map[string]string{
			"hostname": machine.Name,
		},
		StackScriptID:  scriptID,
		Image:          conn.Cluster.ClusterConfig().Cloud.InstanceImage,
		BackupsEnabled: false,
		PrivateIP:      true,
		SwapSize:       types.IntP(0),
	}

	instance, err := conn.client.CreateInstance(context.Background(), createOpts)
	if err != nil {
		return nil, err
	}

	if err := conn.waitForStatus(instance.ID, linodego.InstanceRunning); err != nil {
		return nil, err
	}

	node := api.NodeInfo{
		Name:       instance.Label,
		ExternalID: strconv.Itoa(instance.ID),
		PublicIP:   instance.IPv4[0].String(),
		PrivateIP:  instance.IPv4[1].String(),
	}
	return &node, nil
}

func (conn *cloudConnector) DeleteInstanceByProviderID(providerID string) error {
	id, err := instanceIDFromProviderID(providerID)
	if err != nil {
		return err
	}

	if err := conn.client.DeleteInstance(context.Background(), id); err != nil {
		return err
	}

	conn.Logger.Info("Instance deleted", "instance-id", id)
	return nil
}

func instanceIDFromProviderID(providerID string) (int, error) {
	if providerID == "" {
		return 0, errors.New("providerID cannot be empty string")
	}

	split := strings.Split(providerID, "/")
	if len(split) != 3 {
		return 0, errors.Errorf("unexpected providerID format: %s, format should be: linode://12345", providerID)
	}

	// since split[0] is actually "linode:"
	if strings.TrimSuffix(split[0], ":") != UID {
		return 0, errors.Errorf("provider name from providerID should be linode: %s", providerID)
	}

	return strconv.Atoi(split[2])
}

func (conn *cloudConnector) instanceIfExists(machine *clusterv1.Machine) (*linodego.Instance, error) {
	lds, err := conn.client.ListInstances(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	for _, ld := range lds {
		if ld.Label == machine.Name {
			return &ld, nil
		}
	}

	return nil, fmt.Errorf("no vm found with %v name", machine.Name)
}

func (conn *cloudConnector) createLoadBalancer(name string) (*linodego.NodeBalancer, error) {
	lb, err := conn.lbByName(name)
	if err != nil {
		if err == errLBNotFound {
			ip, err := conn.buildLoadBalancerRequest(name)
			if err != nil {
				return nil, err
			}
			return ip, nil
		}
	}
	return lb, nil
}

func (conn *cloudConnector) lbByName(name string) (*linodego.NodeBalancer, error) {
	lbs, err := conn.client.ListNodeBalancers(context.Background(), nil)
	if err != nil {
		return nil, err
	}

	for _, lb := range lbs {
		if *lb.Label == name {
			return &lb, nil
		}
	}

	return nil, errLBNotFound
}

func (conn *cloudConnector) buildLoadBalancerRequest(lbName string) (*linodego.NodeBalancer, error) {
	lb, err := conn.createNodeBalancer(lbName)
	if err != nil {
		return nil, err
	}

	nb, err := conn.client.GetNodeBalancer(context.Background(), lb)
	if err != nil {
		return nil, err
	}
	if nb == nil {
		return nil, fmt.Errorf("nodebalancer with id %v not found", lb)
	}

	_, err = conn.createNodeBalancerConfig(lb)
	if err != nil {
		return nil, err
	}

	return nb, nil
}

func (conn *cloudConnector) createNodeBalancerConfig(nbID int) (int, error) {
	tr := true
	resp, err := conn.client.CreateNodeBalancerConfig(context.Background(), nbID, linodego.NodeBalancerConfigCreateOptions{
		Port:          api.DefaultKubernetesBindPort,
		Protocol:      linodego.ProtocolTCP,
		Algorithm:     linodego.AlgorithmLeastConn,
		Stickiness:    linodego.StickinessTable,
		Check:         linodego.CheckConnection,
		CheckInterval: 5,
		CheckTimeout:  3,
		CheckAttempts: 10,
		CheckPassive:  &tr,
	})
	if err != nil {
		return -1, err
	}
	return resp.ID, nil
}
func (conn *cloudConnector) addNodeToBalancer(lbName string, nodeName, ip string) error {
	lb, err := conn.lbByName(lbName)
	if err != nil {
		return err
	}

	lbcs, err := conn.client.ListNodeBalancerConfigs(context.Background(), lb.ID, nil)
	if err != nil {
		return err
	}

	_, err = conn.client.CreateNodeBalancerNode(context.Background(), lb.ID, lbcs[0].ID, linodego.NodeBalancerNodeCreateOptions{
		Address: fmt.Sprintf("%v:%v", ip, api.DefaultKubernetesBindPort),
		Label:   nodeName,
		Weight:  100,
		Mode:    linodego.ModeAccept,
	})
	if err != nil {
		return err
	}

	conn.Logger.Info("Added master to the loadbalancer", "master-name", nodeName, "lb-name", lbName)

	return nil
}

func (conn *cloudConnector) createNodeBalancer(name string) (int, error) {
	connThrottle := 20
	resp, err := conn.client.CreateNodeBalancer(context.Background(), linodego.NodeBalancerCreateOptions{
		Label:              &name,
		Region:             conn.Cluster.ClusterConfig().Cloud.Zone,
		ClientConnThrottle: &connThrottle,
	})
	if err != nil {
		return -1, err
	}
	return resp.ID, nil
}

func (conn *cloudConnector) deleteLoadBalancer(lbName string) error {
	lb, err := conn.lbByName(lbName)
	if err != nil {
		return err
	}

	return conn.client.DeleteNodeBalancer(context.Background(), lb.ID)
}
