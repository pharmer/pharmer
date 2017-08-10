package packet

import (
	"strings"
	"time"

	"github.com/appscode/errors"
	"github.com/appscode/pharmer/contexts"
	"github.com/appscode/pharmer/credential"
	"github.com/packethost/packngo"
)

type cloudConnector struct {
	ctx    *contexts.ClusterContext
	client *packngo.Client
}

func NewConnector(ctx *contexts.ClusterContext) (*cloudConnector, error) {
	apiKey, ok := ctx.CloudCredential[credential.PacketApiKey]
	if !ok {
		return nil, errors.New().WithMessagef("Cluster %v credential is missing %v", ctx.Name, credential.PacketApiKey)
	}
	return &cloudConnector{
		ctx:    ctx,
		client: packngo.NewClient("", apiKey, nil),
	}, nil
}

func (conn *cloudConnector) waitForInstance(deviceID, status string) error {
	attempt := 0
	for true {
		conn.ctx.Logger.Infof("Checking status of instance %v", deviceID)
		s, _, err := conn.client.Devices.Get(deviceID)
		if err != nil {
			return errors.FromErr(err).WithContext(conn.ctx).Err()
		}
		if strings.ToLower(s.State) == status {
			break
		}
		conn.ctx.Logger.Infof("Instance %v (%v) is %v, waiting...", s.Hostname, s.ID, s.State)
		attempt += 1
		time.Sleep(30 * time.Second)
	}
	return nil
}
