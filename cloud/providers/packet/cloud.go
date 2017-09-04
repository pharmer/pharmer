package packet

import (
	"context"
	"strings"
	"time"

	"github.com/appscode/go/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/credential"
	"github.com/packethost/packngo"
)

type cloudConnector struct {
	ctx     context.Context
	cluster *api.Cluster
	client  *packngo.Client
}

func NewConnector(ctx context.Context, cluster *api.Cluster) (*cloudConnector, error) {
	cred, err := cloud.Store(ctx).Credentials().Get(cluster.Spec.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.Packet{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.New().WithMessagef("Credential %s is invalid. Reason: %v", cluster.Spec.CredentialName, err)
	}
	// TODO: FixIt Project ID
	cluster.Spec.Project = typed.ProjectID()
	return &cloudConnector{
		ctx:    ctx,
		client: packngo.NewClient("", typed.APIKey(), nil),
	}, nil
}

func (conn *cloudConnector) waitForInstance(deviceID, status string) error {
	attempt := 0
	for true {
		cloud.Logger(conn.ctx).Infof("Checking status of instance %v", deviceID)
		s, _, err := conn.client.Devices.Get(deviceID)
		if err != nil {
			return errors.FromErr(err).WithContext(conn.ctx).Err()
		}
		if strings.ToLower(s.State) == status {
			break
		}
		cloud.Logger(conn.ctx).Infof("Instance %v (%v) is %v, waiting...", s.Hostname, s.ID, s.State)
		attempt += 1
		time.Sleep(30 * time.Second)
	}
	return nil
}
