package linode

import (
	"context"
	"net/http"

	"github.com/pharmer/cloud/pkg/apis"

	"github.com/linode/linodego"
	v1 "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	"golang.org/x/oauth2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Client struct {
	Client *linodego.Client
}

func NewClient(opts Options) (*Client, error) {
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: opts.Token})

	oauth2Client := &http.Client{
		Transport: &oauth2.Transport{
			Source: tokenSource,
		},
	}

	client := linodego.NewClient(oauth2Client)
	g := &Client{
		Client: &client,
	}
	return g, nil
}

func (g *Client) GetName() string {
	return apis.Linode
}

func (g *Client) ListCredentialFormats() []v1.CredentialFormat {
	return []v1.CredentialFormat{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: apis.Linode,
				Annotations: map[string]string{
					"cloud.pharmer.io/cluster-credential": "",
					"cloud.pharmer.io/dns-credential":     "",
				},
			},
			Spec: v1.CredentialFormatSpec{
				Provider:      apis.Linode,
				DisplayFormat: "field",
				Fields: []v1.CredentialField{
					{
						Envconfig: "LINODE_TOKEN",
						Form:      "linode_token",
						JSON:      "token",
						Label:     "Token",
						Input:     "password",
					},
				},
			},
		},
	}
}

//DataCenter as region
func (g *Client) ListRegions() ([]v1.Region, error) {
	regionList, err := g.Client.ListRegions(context.Background(), &linodego.ListOptions{})
	if err != nil {
		return nil, err
	}
	var regions []v1.Region
	for _, r := range regionList {
		region := ParseRegion(&r)
		regions = append(regions, *region)
	}
	return regions, nil
}

//data.Region.Region as Zone
func (g *Client) ListZones() ([]string, error) {
	regionList, err := g.ListRegions()
	if err != nil {
		return nil, err
	}
	var zones []string
	for _, r := range regionList {
		zones = append(zones, r.Region)
	}
	return zones, nil
}

func (g *Client) ListMachineTypes() ([]v1.MachineType, error) {
	instanceList, err := g.Client.ListTypes(context.Background(), &linodego.ListOptions{})
	if err != nil {
		return nil, err
	}
	var instances []v1.MachineType
	for _, ins := range instanceList {
		instance, err := ParseInstance(&ins)
		if err != nil {
			return nil, err
		}
		instances = append(instances, *instance)
	}
	return instances, nil
}
