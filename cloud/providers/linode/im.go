package linode

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/appscode/data"
	"github.com/appscode/go/errors"
	"github.com/appscode/linodego"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/phid"
	"github.com/cenkalti/backoff"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type instanceManager struct {
	ctx     context.Context
	cluster *api.Cluster
	conn    *cloudConnector
	namer   namer
}

const (
	LinodeStatus_BeingCreated = -1
	LinodeStatus_BrandNew     = 0
	LinodeStatus_Running      = 1
	LinodeStatus_PoweredOff   = 2
)

func (im *instanceManager) GetInstance(md *api.InstanceStatus) (*api.Instance, error) {
	master := net.ParseIP(md.Name) == nil

	var instance *api.Instance
	backoff.Retry(func() error {
		resp, err := im.conn.client.Ip.List(0, 0)
		if err != nil {
			return err
		}
		for _, fip := range resp.FullIPAddresses {
			if fip.IsPublic == 0 && fip.IPAddress == md.PrivateIP {
				linodes, err := im.conn.client.Linode.List(fip.LinodeId)
				if err != nil {
					return err
				}
				instance, err = im.newKubeInstance(&linodes.Linodes[0])
				if err != nil {
					return err
				}
				if master {
					instance.Spec.Role = api.RoleKubernetesMaster
				} else {
					instance.Name = im.cluster.Name + "-node-" + strconv.Itoa(fip.LinodeId)
					instance.Spec.Role = api.RoleKubernetesPool
				}
				return nil
			}
		}
		return nil
	}, backoff.NewExponentialBackOff())

	if instance == nil {
		return nil, errors.New("No instance found with name", md.Name).WithContext(im.ctx).Err()
	}
	return instance, nil
}

func (im *instanceManager) createStackScript(sku, role string) (int, error) {
	startupScript, err := RenderStartupScript(im.ctx, im.cluster, role)
	if err != nil {
		return 0, err
	}
	script, err := im.conn.client.StackScript.Create(im.namer.StartupScriptName(sku, role), im.cluster.Spec.InstanceImage, startupScript, map[string]string{
		"Description": im.cluster.Name,
	})
	if err != nil {
		return 0, err
	}
	cloud.Logger(im.ctx).Infof("Stack script for role %v created", role)
	return script.StackScriptId.StackScriptId, nil
}

func (im *instanceManager) createInstance(name string, scriptId int, sku string) (int, int, error) {
	dcId, err := strconv.Atoi(im.cluster.Spec.Zone)
	if err != nil {
		return 0, 0, errors.FromErr(err).WithContext(im.ctx).Err()
	}
	planId, err := strconv.Atoi(sku)
	if err != nil {
		return 0, 0, errors.FromErr(err).WithContext(im.ctx).Err()
	}
	linode, err := im.conn.client.Linode.Create(dcId, planId, 0)
	if err != nil {
		return 0, 0, errors.FromErr(err).WithContext(im.ctx).Err()
	}
	id := linode.LinodeId.LinodeId

	_, err = im.conn.client.Ip.AddPrivate(id)
	if err != nil {
		return 0, 0, errors.FromErr(err).WithContext(im.ctx).Err()
	}

	_, err = im.conn.client.Linode.Update(id, map[string]interface{}{
		"watchdog": false,
	})
	if err != nil {
		return 0, 0, errors.FromErr(err).WithContext(im.ctx).Err()
	}

	stackScriptUDFResponses := fmt.Sprintf(`{
  "cluster": "%v",
  "instance": "%v",
  "stack_script_id": "%v"
}`, im.cluster.Name, name, scriptId)
	args := map[string]string{
		"rootSSHKey": string(cloud.SSHKey(im.ctx).PublicKey),
	}

	mt, err := data.ClusterMachineType("linode", sku)
	if err != nil {
		return 0, 0, errors.FromErr(err).WithContext(im.ctx).Err()
	}
	distributionID, err := strconv.Atoi(im.cluster.Spec.InstanceImage)
	if err != nil {
		return 0, 0, errors.FromErr(err).WithContext(im.ctx).Err()
	}
	swapDiskSize := 512                // MB
	rootDiskSize := mt.Disk*1024 - 512 // MB
	rootDisk, err := im.conn.client.Disk.CreateFromStackscript(scriptId, id, name, stackScriptUDFResponses, distributionID, rootDiskSize, im.cluster.Spec.InstanceRootPassword, args)
	if err != nil {
		return 0, 0, errors.FromErr(err).WithContext(im.ctx).Err()
	}
	swapDisk, err := im.conn.client.Disk.Create(id, "swap", "swap-disk", swapDiskSize, nil)
	if err != nil {
		return 0, 0, errors.FromErr(err).WithContext(im.ctx).Err()
	}

	kernelId, err := strconv.Atoi(im.cluster.Spec.Kernel)
	if err != nil {
		return 0, 0, errors.FromErr(err).WithContext(im.ctx).Err()
	}
	args = map[string]string{
		"RootDeviceNum": "1",
		"DiskList":      fmt.Sprintf("%d,%d", rootDisk.DiskJob.DiskId, swapDisk.DiskJob.DiskId),
	}
	config, err := im.conn.client.Config.Create(id, kernelId, name, args)
	if err != nil {
		return 0, 0, errors.FromErr(err).WithContext(im.ctx).Err()
	}

	jobResp, err := im.conn.client.Linode.Boot(id, config.LinodeConfigId.LinodeConfigId)
	if err != nil {
		return 0, 0, errors.FromErr(err).WithContext(im.ctx).Err()
	}
	cloud.Logger(im.ctx).Info("Running linode boot job %v", jobResp.JobId.JobId)
	cloud.Logger(im.ctx).Infof("Linode %v created", name)

	return id, config.LinodeConfigId.LinodeConfigId, err
}

func (im *instanceManager) bootToGrub2(linodeId, configId int, name string) error {
	// GRUB2 Kernel ID = 210
	_, err := im.conn.client.Config.Update(configId, linodeId, 210, nil)
	if err != nil {
		return err
	}
	_, err = im.conn.client.Linode.Update(linodeId, map[string]interface{}{
		"Label":    name,
		"watchdog": true,
	})
	if err != nil {
		return err
	}
	_, err = im.conn.client.Linode.Boot(linodeId, configId)
	cloud.Logger(im.ctx).Infof("%v booted", name)
	return err
}

func (im *instanceManager) newKubeInstance(linode *linodego.Linode) (*api.Instance, error) {
	var externalIP, internalIP string
	ips, err := im.conn.client.Ip.List(linode.LinodeId, -1)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(im.ctx).Err()
	}
	for _, ip := range ips.FullIPAddresses {
		if ip.IsPublic == 1 {
			externalIP = ip.IPAddress
		} else {
			internalIP = ip.IPAddress
		}
		if externalIP != "" && internalIP != "" {
			i := api.Instance{
				ObjectMeta: metav1.ObjectMeta{
					UID:  phid.NewKubeInstance(),
					Name: linode.Label.String(),
				},
				Spec: api.InstanceSpec{
					SKU: strconv.Itoa(linode.PlanId),
				},
				Status: api.InstanceStatus{
					ExternalID:    strconv.Itoa(linode.LinodeId),
					PublicIP:      externalIP,
					PrivateIP:     internalIP,
					Phase:         api.InstancePhaseReady,
					ExternalPhase: statusString(linode.Status),
				},
			}
			return &i, nil
		}
	}
	return nil, errors.New("Failed to detect Public IP").WithContext(im.ctx).Err()
}
