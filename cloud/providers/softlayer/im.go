package softlayer

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/appscode/data"
	"github.com/appscode/go/errors"
	. "github.com/appscode/go/types"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/phid"
	"github.com/cenkalti/backoff"
	"github.com/softlayer/softlayer-go/datatypes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type instanceManager struct {
	ctx     context.Context
	cluster *api.Cluster
	conn    *cloudConnector
}

func (im *instanceManager) GetInstance(md *api.InstanceStatus) (*api.Instance, error) {
	master := net.ParseIP(md.Name) == nil
	var instance *api.Instance
	backoff.Retry(func() (err error) {
		for {
			servers, err := im.conn.accountServiceClient.GetVirtualGuests()
			if err != nil {
				return err
			}
			for _, s := range servers {
				interIp := strings.Trim(*s.PrimaryBackendIpAddress, `"`)
				if interIp == md.PrivateIP {
					instance, err = im.newKubeInstance(*s.Id)
					sku := strconv.Itoa(*s.MaxCpu) + "c" + strconv.Itoa(*s.MaxMemory) + "m"
					instance.Spec.SKU = sku
					if err != nil {
						return err
					}
					if master {
						instance.Spec.Role = api.RoleKubernetesMaster
					} else {
						instance.Spec.Role = api.RoleKubernetesPool
					}
					return nil
				}

			}
		}
		return nil
	}, backoff.NewExponentialBackOff())

	if instance == nil {
		return nil, errors.New("No instance found with name", md.Name).WithContext(im.ctx).Err()
	}
	return instance, nil
}

func (im *instanceManager) createInstance(name, role, sku string) (int, error) {
	startupScript, err := cloud.RenderStartupScript(im.ctx, im.cluster, role)
	if err != nil {
		im.cluster.Status.Reason = err.Error()
		return 0, errors.FromErr(err).WithContext(im.ctx).Err()
	}
	instance, err := data.ClusterMachineType(im.cluster.Spec.Cloud.CloudProvider, sku)
	if err != nil {
		im.cluster.Status.Reason = err.Error()
		return 0, errors.FromErr(err).WithContext(im.ctx).Err()
	}
	cpu := instance.CPU
	ram := 0
	switch instance.RAM.(type) {
	case int, int32, int64:
		ram = instance.RAM.(int) * 1024
	case float64, float32:
		ram = int(instance.RAM.(float64) * 1024)
	default:
		return 0, fmt.Errorf("Failed to parse memory metadata for sku %v", sku)
	}

	sshid, err := strconv.Atoi(im.cluster.Status.SSHKeyExternalID)
	if err != nil {
		im.cluster.Status.Reason = err.Error()
		return 0, errors.FromErr(err).WithContext(im.ctx).Err()
	}
	vGuestTemplate := datatypes.Virtual_Guest{
		Hostname:                     StringP(name),
		Domain:                       StringP(cloud.Extra(im.ctx).ExternalDomain(im.cluster.Name)),
		MaxMemory:                    IntP(ram),
		StartCpus:                    IntP(cpu),
		Datacenter:                   &datatypes.Location{Name: StringP(im.cluster.Spec.Cloud.Zone)},
		OperatingSystemReferenceCode: StringP(im.cluster.Spec.Cloud.OS),
		LocalDiskFlag:                TrueP(),
		HourlyBillingFlag:            TrueP(),
		SshKeys: []datatypes.Security_Ssh_Key{
			{
				Id:          IntP(sshid),
				Fingerprint: StringP(cloud.SSHKey(im.ctx).OpensshFingerprint),
			},
		},
		UserData: []datatypes.Virtual_Guest_Attribute{
			{
				//https://sldn.softlayer.com/blog/jarteche/getting-started-user-data-and-post-provisioning-scripts
				Type: &datatypes.Virtual_Guest_Attribute_Type{
					Keyname: StringP("USER_DATA"),
					Name:    StringP("User Data"),
				},
				Value: StringP(startupScript),
			},
		},
		PostInstallScriptUri: StringP("https://raw.githubusercontent.com/appscode/pharmer/master/cloud/providers/softlayer/startupscript.sh"),
	}

	vGuest, err := im.conn.virtualServiceClient.Mask("id;domain").CreateObject(&vGuestTemplate)
	if err != nil {
		im.cluster.Status.Reason = err.Error()
		return 0, errors.FromErr(err).WithContext(im.ctx).Err()
	}
	cloud.Logger(im.ctx).Infof("Softlayer instance %v created", name)
	return *vGuest.Id, nil
}

func (im *instanceManager) newKubeInstance(id int) (*api.Instance, error) {
	bluemix := im.conn.virtualServiceClient.Id(id)
	status, err := bluemix.GetStatus()
	if err != nil {
		return nil, errors.FromErr(err).WithContext(im.ctx).Err()
	}
	d, err := bluemix.GetObject()
	if err != nil {
		return nil, errors.FromErr(err).WithContext(im.ctx).Err()
	}
	ki := &api.Instance{
		ObjectMeta: metav1.ObjectMeta{
			UID:  phid.NewKubeInstance(),
			Name: *d.FullyQualifiedDomainName,
		},
		Status: api.InstanceStatus{
			ExternalID:    strconv.Itoa(id),
			ExternalPhase: *status.Name,
			Phase:         api.InstancePhaseReady, // droplet.Status == active
		},
	}

	ki.Status.PublicIP, err = bluemix.GetPrimaryIpAddress()
	if err != nil {
		return nil, errors.FromErr(err).WithContext(im.ctx).Err()
	}
	ki.Status.PublicIP = strings.Trim(ki.Status.PublicIP, `"`)
	ki.Status.PrivateIP, err = bluemix.GetPrimaryBackendIpAddress()
	if err != nil {
		return nil, errors.FromErr(err).WithContext(im.ctx).Err()
	}
	ki.Status.PrivateIP = strings.Trim(ki.Status.PrivateIP, `"`)

	return ki, nil
}

func (im *instanceManager) reboot(id int) (bool, error) {
	service := im.conn.virtualServiceClient.Id(id)
	return service.RebootDefault()
}
