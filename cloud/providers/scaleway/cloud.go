package scaleway

import (
	"context"
	"strings"
	"time"

	"github.com/appscode/go/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/credential"
	sapi "github.com/scaleway/scaleway-cli/pkg/api"
)

type cloudConnector struct {
	ctx          context.Context
	cluster      *api.Cluster
	client       *sapi.ScalewayAPI
	bootscriptID string
}

func NewConnector(ctx context.Context, cluster *api.Cluster) (*cloudConnector, error) {
	cred, err := cloud.Store(ctx).Credentials().Get(cluster.Spec.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.Scaleway{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.New().WithMessagef("Credential %s is invalid. Reason: %v", cluster.Spec.CredentialName, err)
	}

	client, err := sapi.NewScalewayAPI(typed.Organization(), typed.Token(), "pharmer", cluster.Spec.Zone)
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
					if li.Arch == "x86_64" && li.Zone == conn.cluster.Spec.Zone {
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
		cloud.Logger(conn.ctx).Infof("Checking status of instance %v", id)
		s, err := conn.client.GetServer(id)
		if err != nil {
			return errors.FromErr(err).WithContext(conn.ctx).Err()
		}
		if strings.ToLower(s.State) == status {
			break
		}
		cloud.Logger(conn.ctx).Infof("Instance %v (%v) is %v, waiting...", s.Name, s.Identifier, s.State)
		attempt += 1
		time.Sleep(30 * time.Second)
	}
	return nil
}
