package linode

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	api "github.com/pharmer/pharmer/apis/v1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/credential"
	"github.com/pharmer/pharmer/data/files"
	"github.com/pkg/errors"
	"github.com/taoh/linodego"
	"k8s.io/apimachinery/pkg/util/wait"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

var errLBNotFound = errors.New("loadbalancer not found")

type cloudConnector struct {
	ctx     context.Context
	cluster *api.Cluster
	client  *linodego.Client
	namer   namer
}

func NewConnector(ctx context.Context, cluster *api.Cluster) (*cloudConnector, error) {
	cred, err := Store(ctx).Credentials().Get(cluster.Spec.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.Linode{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.Errorf("credential %s is invalid. Reason: %v", cluster.Spec.CredentialName, err)
	}

	namer := namer{cluster: cluster}
	c := linodego.NewClient(typed.APIToken(), nil)
	c.UsePost = true
	return &cloudConnector{
		ctx:     ctx,
		cluster: cluster,
		namer:   namer,
		client:  c,
	}, nil
}

func (cm *ClusterManager) PrepareCloud(clusterName string) error {
	var err error
	cluster, err := Store(cm.ctx).Clusters().Get(clusterName)
	if err != nil {
		return fmt.Errorf("cluster `%s` does not exist. Reason: %v", clusterName, err)
	}
	cm.cluster = cluster

	if cm.ctx, err = LoadCACertificates(cm.ctx, cm.cluster); err != nil {
		return err
	}
	if cm.ctx, err = LoadEtcdCertificate(cm.ctx, cm.cluster); err != nil {
		return err
	}
	if cm.ctx, err = LoadSSHKey(cm.ctx, cm.cluster); err != nil {
		return err
	}
	if cm.ctx, err = LoadSaKey(cm.ctx, cm.cluster); err != nil {
		return err
	}
	if cm.conn, err = NewConnector(cm.ctx, cm.cluster); err != nil {
		return err
	}
	cm.namer = namer{cluster: cm.cluster}
	return nil
}

func (conn *cloudConnector) DetectInstanceImage() (string, error) {
	resp, err := conn.client.Avail.Distributions()
	if err != nil {
		return "", err
	}
	for _, d := range resp.Distributions {
		if d.Is64Bit == 1 && d.Label.String() == "Ubuntu 16.04 LTS" {
			return strconv.Itoa(d.DistributionId), nil
		}
	}
	return "", errors.New("can't find `Ubuntu 16.04 LTS` image")
}

func (conn *cloudConnector) DetectKernel() (int64, error) {
	resp, err := conn.client.Avail.Kernels(map[string]string{
		"isKVM": "true",
	})
	if err != nil {
		return 0, err
	}
	kernelId := -1
	for _, d := range resp.Kernels {
		if d.IsPVOPS == 1 {
			if strings.HasPrefix(d.Label.String(), "Latest 64 bit") {
				return int64(d.KernelId), nil
			}
			if strings.Contains(d.Label.String(), "x86_64") && d.KernelId > kernelId {
				kernelId = d.KernelId
			}
		}
	}
	if kernelId >= 0 {
		return int64(kernelId), nil
	}
	return 0, errors.New("can't find Kernel")
}

// ---------------------------------------------------------------------------------------------------------------------

func (conn *cloudConnector) waitForStatus(id, status int) error {
	attempt := 0
	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		attempt++

		resp, err := conn.client.Linode.List(id)
		if err != nil {
			return false, nil
		}
		if len(resp.Linodes) == 0 {
			return false, nil
		}
		server := resp.Linodes[0]
		Logger(conn.ctx).Infof("Attempt %v: Instance `%v` is in status `%s`", attempt, id, statusString(server.Status))
		if server.Status == status {
			return true, nil
		}
		return false, nil
	})
}

/*
Status values are -1: Being Created, 0: Brand New, 1: Running, and 2: Powered Off.
*/
func statusString(status int) string {
	switch status {
	case LinodeStatus_BeingCreated:
		return "Being Created"
	case LinodeStatus_BrandNew:
		return "Brand New"
	case LinodeStatus_Running:
		return "Running"
	case LinodeStatus_PoweredOff:
		return "Powered Off"
	default:
		return strconv.Itoa(status)
	}
}

const (
	LinodeStatus_BeingCreated = -1
	LinodeStatus_BrandNew     = 0
	LinodeStatus_Running      = 1
	LinodeStatus_PoweredOff   = 2
)

// ---------------------------------------------------------------------------------------------------------------------

func (conn *cloudConnector) getStartupScriptID(machineConf *api.MachineProviderConfig, role string) (int, error) {
	scriptName := conn.namer.StartupScriptName(machineConf.Config.SKU, role)
	scripts, err := conn.client.StackScript.List(0)
	if err != nil {
		return 0, err
	}
	for _, s := range scripts.StackScripts {
		if s.Label.String() == scriptName {
			return s.StackScriptId, nil
		}
	}
	return 0, ErrNotFound
}

func (conn *cloudConnector) instanceIfExists(machine *clusterv1.Machine) (*linodego.Linode, error) {
	linodes, err := conn.client.Linode.List(0)
	if err != nil {
		return nil, err
	}
	for _, lin := range linodes.Linodes {
		if lin.Label.String() == machine.Name {
			l, err := conn.client.Linode.List(lin.LinodeId)
			if err != nil {
				return nil, err
			}
			return &l.Linodes[0], nil
		}
	}

	return nil, fmt.Errorf("no droplet found with %v name", machine.Name)
}

func (conn *cloudConnector) createOrUpdateStackScript(machine *clusterv1.Machine, token string) (int, error) {
	machineConf, err := conn.cluster.MachineProviderConfig(machine)
	if err != nil {
		return 0, err
	}

	scriptName := conn.namer.StartupScriptName(machineConf.Config.SKU, string(machine.Spec.Roles[0]))
	script, err := conn.renderStartupScript(conn.cluster, machine, token)
	if err != nil {
		return 0, err
	}

	scripts, err := conn.client.StackScript.List(0)
	if err != nil {
		return 0, err
	}
	for _, s := range scripts.StackScripts {
		if s.Label.String() == scriptName {
			resp, err := conn.client.StackScript.Update(s.StackScriptId, map[string]string{
				"script": script,
			})
			if err != nil {
				return 0, err
			}
			//Logger(conn.ctx).Infof("Stack script for role %v updated", ng.Role())
			return resp.StackScriptId.StackScriptId, nil
		}
	}

	resp, err := conn.client.StackScript.Create(scriptName, conn.cluster.ProviderConfig().InstanceImage, script, map[string]string{
		"Description": fmt.Sprintf("Startup script for NodeGroup %s of Cluster %s", machine.Name, conn.cluster.Name),
	})
	if err != nil {
		return 0, err
	}
	//	Logger(conn.ctx).Infof("Stack script for role %v created", ng.Role())
	return resp.StackScriptId.StackScriptId, nil
}

func (conn *cloudConnector) deleteStackScript(machineConf *api.MachineProviderConfig, role string) error {
	scriptName := conn.namer.StartupScriptName(machineConf.Config.SKU, role)
	scripts, err := conn.client.StackScript.List(0)
	if err != nil {
		return err
	}
	for _, s := range scripts.StackScripts {
		if s.Label.String() == scriptName {
			_, err := conn.client.StackScript.Delete(s.StackScriptId)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// ---------------------------------------------------------------------------------------------------------------------

func (conn *cloudConnector) CreateInstance(cluster *api.Cluster, machine *clusterv1.Machine, token string) (*api.NodeInfo, error) {
	clusterConfig := cluster.ProviderConfig()
	machineConfig, err := cluster.MachineProviderConfig(machine)
	if err != nil {
		return nil, err
	}

	dcId, err := strconv.Atoi(clusterConfig.Zone)
	if err != nil {
		return nil, err
	}
	planId, err := strconv.Atoi(machineConfig.Config.SKU)
	if err != nil {
		return nil, err
	}
	server, err := conn.client.Linode.Create(dcId, planId, 0)
	if err != nil {
		return nil, err
	}
	linodeId := server.LinodeId.LinodeId
	_, err = conn.client.Ip.AddPrivate(linodeId)
	if err != nil {
		return nil, err
	}
	err = conn.waitForStatus(linodeId, LinodeStatus_BrandNew)
	if err != nil {
		return nil, err
	}

	node := api.NodeInfo{
		ExternalID: strconv.Itoa(linodeId),
	}
	ips, err := conn.client.Ip.List(linodeId, -1)
	if err != nil {
		return nil, err
	}
	for _, ip := range ips.FullIPAddresses {
		if ip.IsPublic == 1 {
			node.PublicIP = ip.IPAddress
		} else {
			node.PrivateIP = ip.IPAddress
		}
	}
	parts := strings.SplitN(node.PublicIP, ".", 4)
	node.Name = fmt.Sprintf("%s-%03s-%03s-%03s-%03s", conn.cluster.Name, parts[0], parts[1], parts[2], parts[3])

	_, err = conn.client.Linode.Update(linodeId, map[string]interface{}{
		"Label": node.Name,
	})
	if err != nil {
		return nil, err
	}

	scriptId, err := conn.getStartupScriptID(machineConfig, string(machine.Spec.Roles[0]))
	if err != nil {
		return nil, err
	}

	stackScriptUDFResponses := fmt.Sprintf(`{"hostname": "%s"}`, node.Name)
	mt, err := files.GetInstanceType("linode", machineConfig.Config.SKU)
	if err != nil {
		return nil, err
	}
	distributionID, err := strconv.Atoi(clusterConfig.InstanceImage)
	if err != nil {
		return nil, err
	}
	// swapDiskSize := 512                      // MB
	swapDiskSize := 0                           // https://github.com/kubernetes/kubernetes/issues/53533#issuecomment-335219173
	rootDiskSize := mt.Disk*1024 - swapDiskSize // MB
	rootDisk, err := conn.client.Disk.CreateFromStackscript(
		scriptId,
		linodeId,
		node.Name,
		stackScriptUDFResponses,
		distributionID,
		rootDiskSize,
		clusterConfig.Linode.RootPassword,
		map[string]string{
			"rootSSHKey": string(SSHKey(conn.ctx).PublicKey),
		})
	if err != nil {
		return nil, err
	}
	//swapDisk, err := conn.client.Disk.Create(linodeId, "swap", "swap-disk", swapDiskSize, nil)
	//if err != nil {
	//	return nil, err
	//}
	config, err := conn.client.Config.Create(linodeId, int(clusterConfig.Linode.KernelId), node.Name, map[string]string{
		"RootDeviceNum": "1",
		"DiskList":      strconv.Itoa(rootDisk.DiskJob.DiskId),
		// "DiskList":   fmt.Sprintf("%d,%d", rootDisk.DiskJob.DiskId, swapDisk.DiskJob.DiskId),
	})
	if err != nil {
		return nil, err
	}
	jobResp, err := conn.client.Linode.Boot(linodeId, config.LinodeConfigId.LinodeConfigId)
	if err != nil {
		return nil, err
	}
	Logger(conn.ctx).Infof("Running linode boot job %v", jobResp.JobId.JobId)

	return &node, nil
}

func (conn *cloudConnector) bootToGrub2(linodeId, configId int, name string) error {
	// GRUB2 Kernel ID = 210
	_, err := conn.client.Config.Update(configId, linodeId, 210, nil)
	if err != nil {
		return err
	}
	_, err = conn.client.Linode.Update(linodeId, map[string]interface{}{
		"Label":    name,
		"watchdog": true,
	})
	if err != nil {
		return err
	}
	_, err = conn.client.Linode.Boot(linodeId, configId)
	Logger(conn.ctx).Infof("%v booted", name)
	return err
}

func (conn *cloudConnector) DeleteInstanceByProviderID(providerID string) error {
	id, err := serverIDFromProviderID(providerID)
	if err != nil {
		return err
	}
	_, err = conn.client.Linode.Delete(id, true)
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("Droplet %v deleted", id)
	return nil
}

// dropletIDFromProviderID returns a droplet's ID from providerID.
//
// The providerID spec should be retrievable from the Kubernetes
// node object. The expected format is: digitalocean://droplet-id
// ref: https://github.com/digitalocean/digitalocean-cloud-controller-manager/blob/f9a9856e99c9d382db3777d678f29d85dea25e91/do/droplets.go#L211
func serverIDFromProviderID(providerID string) (int, error) {
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

// ---------------------------------------------------------------------------------------------------------------------

func (conn *cloudConnector) createLoadBalancer(ctx context.Context, name string) (string, error) {
	lb, err := conn.lbByName(name)
	if err != nil {
		if err == errLBNotFound {
			ip, err := conn.buildLoadBalancerRequest(name)
			if err != nil {
				return "", err
			}
			return ip, nil

		}
	}

	err = l.UpdateLoadBalancer(ctx, clusterName, service, nodes)
	if err != nil {
		return nil, err
	}

	lbStatus, exists, err = l.GetLoadBalancer(ctx, clusterName, service)
	if err != nil {
		return nil, err
	}

}

func (conn *cloudConnector) lbByName(name string) (*linodego.LinodeNodeBalancer, error) {
	lbs, err := conn.client.NodeBalancer.List(0)
	if err != nil {
		return nil, err
	}

	for _, lb := range lbs.NodeBalancer {
		if lb.Label.String() == name {
			return &lb, nil
		}
	}

	return nil, errLBNotFound
}

func (conn *cloudConnector) buildLoadBalancerRequest(lbName string) (string, error) {
	lb, err := conn.createNoadBalancer(lbName)
	if err != nil {
		return "", err
	}

	nb, err := conn.client.NodeBalancer.List(lb)
	if err != nil {
		return "", err
	}
	if len(nb.NodeBalancer) == 0 {
		return "", fmt.Errorf("nodebalancer with id %v not found", lb)
	}

	_, err = conn.createNodeBalancerConfig(lb)
	if err != nil {
		return "", err
	}

	/*for _, node := range nodes {
		if err = createNBNode(l.client, ncid, node, port.NodePort); err != nil {
			return "", err
		}
	}*/

	return nb.NodeBalancer[0].Address4, nil
}

func (conn *cloudConnector) createNodeBalancerConfig(nbId int) (int, error) {
	args := map[string]string{
		"Port":       "6443",
		"Protocol":   "tcp",
		"Algorithm":  "leastconn",
		"Stickiness": "table",
	}

	healthArgs := map[string]string{
		"check":          "connection",
		"check_interval": "5",
		"check_timeout":  "3",
		"check_attempts": "2",
		"check_passive":  "true",
	}
	/*if health == "http" || health == "http_body" {
		path := service.Annotations[annLinodeCheckPath]
		if path == "" {
			path = "/"
		}
		args["check_path"] = path
	}
	*/

	args = mergeMaps(args, healthArgs)

	/*tlsArgs, err := getTLSArgs(service, port, protocol)
	if err != nil {
		return -1, err
	}
	args = mergeMaps(args, tlsArgs)*/
	resp, err := conn.client.NodeBalancerConfig.Create(nbId, args)
	if err != nil {
		return -1, err
	}
	return resp.NodeBalancerConfigId.NodeBalancerConfigId, nil
}
func (conn *cloudConnector) createNoadBalancer(name string) (int, error) {
	did, err := strconv.Atoi(conn.cluster.ProviderConfig().Zone)
	if err != nil {
		return -1, err
	}

	resp, err := conn.client.NodeBalancer.Create(did, name, nil)
	if err != nil {
		return -1, err
	}
	return resp.NodeBalancerId.NodeBalancerId, nil
}

func mergeMaps(first, second map[string]string) map[string]string {
	for k, v := range second {
		first[k] = v
	}
	return first
}
