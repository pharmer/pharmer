package scaleway

import (
	"strings"
	"time"

	"github.com/appscode/errors"
	"github.com/appscode/pharmer/contexts"
	"github.com/appscode/pharmer/util/credentialutil"
	sapi "github.com/scaleway/scaleway-cli/pkg/api"
)

type cloudConnector struct {
	ctx          *contexts.ClusterContext
	client       *sapi.ScalewayAPI
	bootscriptID string
}

func NewConnector(ctx *contexts.ClusterContext) (*cloudConnector, error) {
	organization, ok := ctx.CloudCredential[credentialutil.ScalewayCredentialOrganization]
	if !ok {
		return nil, errors.New().WithMessagef("Cluster %v credential is missing %v", ctx.Name, credentialutil.ScalewayCredentialOrganization)
	}
	token, ok := ctx.CloudCredential[credentialutil.ScalewayCredentialToken]
	if !ok {
		return nil, errors.New().WithMessagef("Cluster %v credential is missing %v", ctx.Name, credentialutil.ScalewayCredentialToken)
	}

	client, err := sapi.NewScalewayAPI(organization, token, "appscode", ctx.Zone)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(ctx).Err()
	}
	return &cloudConnector{
		ctx:    ctx,
		client: client,
	}, nil
}

func (conn *cloudConnector) getInstanceImage() (string, error) {
	imgs, err := conn.client.GetMarketPlaceImages("")
	if err != nil {
		return "", errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	for _, img := range imgs.Images {
		if img.Name == "Debian Jessie" {
			for _, v := range img.Versions {
				for _, li := range v.LocalImages {
					if li.Arch == "x86_64" && li.Zone == conn.ctx.Zone {
						return li.ID, nil
					}
				}
			}
		}
	}
	return "", errors.New("Debian Jessie not found for Scaleway").WithContext(conn.ctx).Err()
}

// http://devhub.scaleway.com/#/bootscripts
func (conn *cloudConnector) DetectBootscript() error {
	scripts, err := conn.client.GetBootscripts()
	if err != nil {
		return errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	for _, s := range *scripts {
		// x86_64 4.8.3 docker #1
		if s.Arch == "x86_64" && strings.Contains(s.Title, "docker") {
			conn.bootscriptID = s.Identifier
			return nil
		}
	}
	return errors.New("Docker bootscript not found for Scaleway").WithContext(conn.ctx).Err()
}

func (conn *cloudConnector) waitForInstance(id, status string) error {
	attempt := 0
	for true {
		conn.ctx.Logger.Infof("Checking status of instance %v", id)
		s, err := conn.client.GetServer(id)
		if err != nil {
			return errors.FromErr(err).WithContext(conn.ctx).Err()
		}
		if strings.ToLower(s.State) == status {
			break
		}
		conn.ctx.Logger.Infof("Instance %v (%v) is %v, waiting...", s.Name, s.Identifier, s.State)
		attempt += 1
		time.Sleep(30 * time.Second)
	}
	return nil
}
