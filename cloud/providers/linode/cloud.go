package linode

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/appscode/data"
	api "github.com/appscode/pharmer/apis/v1alpha1"
	. "github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/credential"
	"github.com/taoh/linodego"
	"k8s.io/apimachinery/pkg/util/wait"
)

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
		return nil, fmt.Errorf("credential %s is invalid. Reason: %v", cluster.Spec.CredentialName, err)
	}
	c := linodego.NewClient(typed.APIToken(), nil)
	c.UsePost = true
	return &cloudConnector{
		ctx:     ctx,
		cluster: cluster,
		namer:   namer{cluster: cluster},
		client:  c,
	}, nil
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

func (conn *cloudConnector) getStartupScriptID(ng *api.NodeGroup) (int, error) {
	scriptName := conn.namer.StartupScriptName(ng.Name, ng.Role())
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

func (conn *cloudConnector) createOrUpdateStackScript(ng *api.NodeGroup, token string) (int, error) {
	scriptName := conn.namer.StartupScriptName(ng.Name, ng.Role())
	script, err := conn.renderStartupScript(ng, token)
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
			Logger(conn.ctx).Infof("Stack script for role %v updated", ng.Role())
			return resp.StackScriptId.StackScriptId, nil
		}
	}

	resp, err := conn.client.StackScript.Create(scriptName, conn.cluster.Spec.Cloud.InstanceImage, script, map[string]string{
		"Description": fmt.Sprintf("Startup script for NodeGroup %s of Cluster %s", ng.Name, conn.cluster.Name),
	})
	if err != nil {
		return 0, err
	}
	Logger(conn.ctx).Infof("Stack script for role %v created", ng.Role())
	return resp.StackScriptId.StackScriptId, nil
}

func (conn *cloudConnector) deleteStackScript(ng *api.NodeGroup) error {
	scriptName := conn.namer.StartupScriptName(ng.Name, ng.Role())
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

func (conn *cloudConnector) CreateInstance(_, token string, ng *api.NodeGroup) (*api.NodeInfo, error) {
	dcId, err := strconv.Atoi(conn.cluster.Spec.Cloud.Zone)
	if err != nil {
		return nil, err
	}
	planId, err := strconv.Atoi(ng.Spec.Template.Spec.SKU)
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

	scriptId, err := conn.getStartupScriptID(ng)
	if err != nil {
		return nil, err
	}

	stackScriptUDFResponses := fmt.Sprintf(`{"hostname": "%s"}`, node.Name)
	mt, err := data.ClusterMachineType("linode", ng.Spec.Template.Spec.SKU)
	if err != nil {
		return nil, err
	}
	distributionID, err := strconv.Atoi(conn.cluster.Spec.Cloud.InstanceImage)
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
		conn.cluster.Spec.Cloud.Linode.RootPassword,
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
	config, err := conn.client.Config.Create(linodeId, int(conn.cluster.Spec.Cloud.Linode.KernelId), node.Name, map[string]string{
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
		return 0, fmt.Errorf("unexpected providerID format: %s, format should be: digitalocean://12345", providerID)
	}

	// since split[0] is actually "digitalocean:"
	if strings.TrimSuffix(split[0], ":") != UID {
		return 0, fmt.Errorf("provider name from providerID should be digitalocean: %s", providerID)
	}

	return strconv.Atoi(split[2])
}

// ---------------------------------------------------------------------------------------------------------------------
