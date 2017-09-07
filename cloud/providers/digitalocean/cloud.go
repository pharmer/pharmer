package digitalocean

import (
	"context"
	gtx "context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/appscode/go/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/credential"
	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
	"k8s.io/apimachinery/pkg/util/wait"
)

const containerOsImage = "appscode-containeros"

type cloudConnector struct {
	ctx     context.Context
	cluster *api.Cluster
	client  *godo.Client
}

func NewConnector(ctx context.Context, cluster *api.Cluster) (*cloudConnector, error) {
	cred, err := cloud.Store(ctx).Credentials().Get(cluster.Spec.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.DigitalOcean{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.New().WithMessagef("Credential %s is invalid. Reason: %v", cluster.Spec.CredentialName, err)
	}
	oauthClient := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: typed.Token(),
	}))
	conn := cloudConnector{
		ctx:     ctx,
		cluster: cluster,
		client:  godo.NewClient(oauthClient),
	}
	if ok, msg := conn.IsUnauthorized(); !ok {
		return nil, fmt.Errorf("Credential `%s` does not have necessary autheorization. Reason: %s.", cluster.Spec.CredentialName, msg)
	}
	return &conn, nil
}

// Returns true if unauthorized
func (conn *cloudConnector) IsUnauthorized() (bool, string) {
	name := "check-write-access:" + strconv.FormatInt(time.Now().Unix(), 10)
	_, _, err := conn.client.Tags.Create(gtx.TODO(), &godo.TagCreateRequest{
		Name: name,
	})
	if err != nil {
		return false, "Credential missing WRITE scope"
	}
	conn.client.Tags.Delete(gtx.TODO(), name)
	return true, ""
}

func (conn *cloudConnector) waitForInstance(id int, status string) error {
	attempt := 0
	return wait.PollImmediate(cloud.RetryInterval, cloud.RetryTimeout, func() (bool, error) {
		attempt++

		droplet, _, err := conn.client.Droplets.Get(gtx.TODO(), id)
		if err != nil {
			return false, nil
		}
		cloud.Logger(conn.ctx).Infof("Attempt %v: Instance `%v` is in status `%s`", attempt, id, droplet.Status)
		if strings.ToLower(droplet.Status) == status {
			return true, nil
		}
		return false, nil
	})
}
