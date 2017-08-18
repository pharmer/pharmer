package packet

import (
	"strings"
	"time"

	"github.com/appscode/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/context"
	"github.com/appscode/pharmer/credential"
	"github.com/packethost/packngo"
)

type cloudConnector struct {
	ctx     context.Context
	cluster *api.Cluster
	client  *packngo.Client
}

func NewConnector(ctx context.Context, cluster *api.Cluster) (*cloudConnector, error) {
	apiKey, ok := cluster.Spec.CloudCredential[credential.PacketApiKey]
	if !ok {
		return nil, errors.New().WithMessagef("Cluster %v credential is missing %v", cluster.Name, credential.PacketApiKey)
	}
	return &cloudConnector{
		ctx:    ctx,
		client: packngo.NewClient("", apiKey, nil),
	}, nil
}

func (conn *cloudConnector) waitForInstance(deviceID, status string) error {
	attempt := 0
	for true {
		conn.ctx.Logger().Infof("Checking status of instance %v", deviceID)
		s, _, err := conn.client.Devices.Get(deviceID)
		if err != nil {
			return errors.FromErr(err).WithContext(conn.ctx).Err()
		}
		if strings.ToLower(s.State) == status {
			break
		}
		conn.ctx.Logger().Infof("Instance %v (%v) is %v, waiting...", s.Hostname, s.ID, s.State)
		attempt += 1
		time.Sleep(30 * time.Second)
	}
	return nil
}
