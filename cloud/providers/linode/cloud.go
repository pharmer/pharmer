package linode

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/appscode/go/errors"
	"github.com/appscode/linodego"
	api "github.com/appscode/pharmer/apis/v1alpha1"
	. "github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/credential"
)

type cloudConnector struct {
	ctx     context.Context
	cluster *api.Cluster
	client  *linodego.Client
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
		ctx:    ctx,
		client: linodego.NewClient(typed.APIToken(), nil),
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

func (conn *cloudConnector) waitForStatus(id, status int) (*linodego.Linode, error) {
	attempt := 0
	for true {
		Logger(conn.ctx).Infof("Checking status of instance %v", id)
		resp, err := conn.client.Linode.List(id)
		if err != nil {
			return nil, errors.FromErr(err).WithContext(conn.ctx).Err()
		}
		linode := resp.Linodes[0]
		Logger(conn.ctx).Debugf("Instance status %v, %v", linode.Status, err)
		if linode.Status == status {
			return &linode, nil
		}
		Logger(conn.ctx).Infof("Instance %v (%v) is %v, waiting...", linode.Label, linode.LinodeId, linode.Status)
		attempt += 1
		if attempt > 4*15 {
			break // timeout after 15 mins
		}
		Logger(conn.ctx).Debugf("Attempt %v to linode %v to become %v", attempt, id, statusString(status))
		time.Sleep(15 * time.Second)
	}
	return nil, errors.New("Time out on waiting for linode status to become", statusString(status)).WithContext(conn.ctx).Err()
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
