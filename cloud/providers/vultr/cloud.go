package vultr

import (
	"context"
	"strconv"
	"strings"
	"time"

	gv "github.com/JamesClonk/vultr/lib"
	"github.com/appscode/go/errors"
	api "github.com/appscode/pharmer/apis/v1alpha1"
	. "github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/credential"
)

type cloudConnector struct {
	ctx     context.Context
	cluster *api.Cluster
	client  *gv.Client
}

func NewConnector(ctx context.Context, cluster *api.Cluster) (*cloudConnector, error) {
	cred, err := Store(ctx).Credentials().Get(cluster.Spec.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.DigitalOcean{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.New().WithMessagef("Credential %s is invalid. Reason: %v", cluster.Spec.CredentialName, err)
	}
	return &cloudConnector{
		ctx:     ctx,
		cluster: cluster,
		client:  gv.NewClient(typed.Token(), &gv.Options{}),
	}, nil
}

func (conn *cloudConnector) detectInstanceImage() error {
	oses, err := conn.client.GetOS()
	if err != nil {
		return errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	for _, os := range oses {
		if os.Arch == "x64" && os.Family == "ubuntu" && strings.HasPrefix(os.Name, "Ubuntu 16.04 x64") {
			conn.cluster.Spec.Cloud.InstanceImage = strconv.Itoa(os.ID)
			return nil
		}
	}

	return errors.New("Can't find Debian 8 image").WithContext(conn.ctx).Err()
}

/*
The "status" field represents the status of the subscription and will be one of:
pending | active | suspended | closed. If the status is "active", you can check "power_status"
to determine if the VPS is powered on or not. When status is "active", you may also use
"server_state" for a more detailed status of: none | locked | installingbooting | isomounting | ok.
*/
func (conn *cloudConnector) waitForActiveInstance(id string) (*gv.Server, error) {
	attempt := 0
	for true {
		Logger(conn.ctx).Infof("Checking status of instance %v", id)
		server, err := conn.client.GetServer(id)
		if err != nil {
			return nil, errors.FromErr(err).WithContext(conn.ctx).Err()
		}
		Logger(conn.ctx).Debugf("Instance status %v, %v", server.Status, err)
		if strings.ToLower(server.Status) == "active" && server.PowerStatus == "running" {
			return &server, nil
		}
		Logger(conn.ctx).Infof("Instance %v (%v) is %v, waiting...", server.Name, server.ID, server.Status)
		attempt += 1
		if attempt > 120 {
			break // timeout = 60 mins
		}
		time.Sleep(30 * time.Second)
	}
	return nil, errors.New("Timed out waiting for instance to become active.").WithContext(conn.ctx).Err()
}

func (conn *cloudConnector) getServer(id string) (*gv.Server, error) {
	server, err := conn.client.GetServer(id)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	return &server, nil
}
