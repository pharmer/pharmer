package linode

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/appscode/data"
	"github.com/appscode/go/errors"
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
		return nil, errors.New().WithMessagef("Credential %s is invalid. Reason: %v", cluster.Spec.CredentialName, err)
	}
	return &cloudConnector{
		ctx:     ctx,
		cluster: cluster,
		namer:   namer{cluster: cluster},
		client:  linodego.NewClient(typed.APIToken(), nil),
	}, nil
}

func (conn *cloudConnector) detectInstanceImage() error {
	resp, err := conn.client.Avail.Distributions()
	if err != nil {
		return errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	Logger(conn.ctx).Infof("Checking for instance image")
	for _, d := range resp.Distributions {
		if d.Is64Bit == 1 && d.Label.String() == "Debian 8" {
			conn.cluster.Spec.Cloud.InstanceImage = strconv.Itoa(d.DistributionId)
			Logger(conn.ctx).Infof("Instance image %v with id %v found", d.Label.String(), conn.cluster.Spec.Cloud.InstanceImage)
			return nil
		}
	}
	return errors.New("Can't find Debian 8 image").WithContext(conn.ctx).Err()
}

func (conn *cloudConnector) detectKernel() error {
	resp, err := conn.client.Avail.Kernels(map[string]string{
		"isKVM": "true",
	})
	if err != nil {
		return errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	kernelId := -1
	for _, d := range resp.Kernels {
		if d.IsPVOPS == 1 {
			if strings.HasPrefix(d.Label.String(), "Latest 64 bit") {
				conn.cluster.Spec.Cloud.Kernel = strconv.Itoa(d.KernelId)
				return nil
			}
			if strings.Contains(d.Label.String(), "x86_64") && d.KernelId > kernelId {
				kernelId = d.KernelId
			}
		}
	}
	if kernelId >= 0 {
		conn.cluster.Spec.Cloud.Kernel = strconv.Itoa(kernelId)
		return nil
	}
	return errors.New("Can't find Kernel").WithContext(conn.ctx).Err()
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
		Logger(conn.ctx).Infof("Attempt %v: Instance `%v` is in status `%s`", attempt, id, server.Status)
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

func (conn *cloudConnector) CreateInstance(name, token string, ng *api.NodeGroup) (*api.NodeInfo, error) {
	dcId, err := strconv.Atoi(conn.cluster.Spec.Cloud.Zone)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	planId, err := strconv.Atoi(ng.Spec.Template.Spec.SKU)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	server, err := conn.client.Linode.Create(dcId, planId, 0)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	linodeId := server.LinodeId.LinodeId

	_, err = conn.client.Ip.AddPrivate(linodeId)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(conn.ctx).Err()
	}

	scriptId, err := conn.getStartupScriptID(ng)
	if err != nil {
		return nil, err
	}

	stackScriptUDFResponses := fmt.Sprintf(`{
  "cluster": "%v",
  "instance": "%v",
  "stack_script_id": "%v"
}`, conn.cluster.Name, name, scriptId)
	args := map[string]string{
		"rootSSHKey": string(SSHKey(conn.ctx).PublicKey),
	}

	mt, err := data.ClusterMachineType("linode", ng.Spec.Template.Spec.SKU)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	distributionID, err := strconv.Atoi(conn.cluster.Spec.Cloud.InstanceImage)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	swapDiskSize := 512                // MB
	rootDiskSize := mt.Disk*1024 - 512 // MB
	rootDisk, err := conn.client.Disk.CreateFromStackscript(scriptId, linodeId, name, stackScriptUDFResponses, distributionID, rootDiskSize, conn.cluster.Spec.Cloud.Linode.InstanceRootPassword, args)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	swapDisk, err := conn.client.Disk.Create(linodeId, "swap", "swap-disk", swapDiskSize, nil)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(conn.ctx).Err()
	}

	kernelId, err := strconv.Atoi(conn.cluster.Spec.Cloud.Kernel)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	args = map[string]string{
		"RootDeviceNum": "1",
		"DiskList":      fmt.Sprintf("%d,%d", rootDisk.DiskJob.DiskId, swapDisk.DiskJob.DiskId),
	}
	config, err := conn.client.Config.Create(linodeId, kernelId, name, args)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(conn.ctx).Err()
	}

	jobResp, err := conn.client.Linode.Boot(linodeId, config.LinodeConfigId.LinodeConfigId)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	Logger(conn.ctx).Info("Running linode boot job %v", jobResp.JobId.JobId)
	Logger(conn.ctx).Infof("Linode %v created", name)

	// return linodeId, config.LinodeConfigId.LinodeConfigId, err

	err = conn.waitForStatus(linodeId, LinodeStatus_Running)
	if err != nil {
		return nil, err
	}

	resp, err := conn.client.Linode.List(linodeId)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	host := resp.Linodes[0]

	node := api.NodeInfo{
		Name:       host.Label.String(),
		ExternalID: strconv.Itoa(host.LinodeId),
	}

	ips, err := conn.client.Ip.List(linodeId, -1)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	for _, ip := range ips.FullIPAddresses {
		if ip.IsPublic == 1 {
			node.PublicIP = ip.IPAddress
		} else {
			node.PrivateIP = ip.IPAddress
		}
	}
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
