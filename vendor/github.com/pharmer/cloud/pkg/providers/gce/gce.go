package gce

import (
	"context"

	"github.com/pharmer/cloud/pkg/apis"
	v1 "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	"github.com/pharmer/cloud/pkg/credential"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
)

type Client struct {
	GceProjectID   string
	ComputeService *compute.Service
	Ctx            context.Context
}

func NewClient(opts credential.GCE) (*Client, error) {
	g := &Client{
		GceProjectID: opts.ProjectID(),
		Ctx:          context.Background(),
	}
	var err error
	g.ComputeService, err = getComputeService(g.Ctx, opts.ServiceAccount())
	if err != nil {
		return nil, err
	}
	return g, nil
}

func (g *Client) GetName() string {
	return apis.GCE
}

func (g *Client) GetCredentialFormat() v1.CredentialFormat {
	return credential.GCE{}.Format()
}

func (g *Client) ListRegions() ([]v1.Region, error) {
	req := g.ComputeService.Regions.List(g.GceProjectID)

	var regions []v1.Region
	err := req.Pages(g.Ctx, func(list *compute.RegionList) error {
		for _, region := range list.Items {
			res, err := ParseRegion(region)
			if err != nil {
				return err
			}
			regions = append(regions, *res)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return regions, err
}

func (g *Client) ListZones() ([]string, error) {
	req := g.ComputeService.Zones.List(g.GceProjectID)
	var zones []string
	err := req.Pages(g.Ctx, func(list *compute.ZoneList) error {
		for _, zone := range list.Items {
			zones = append(zones, zone.Name)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return zones, nil
}

func (g *Client) ListMachineTypes() ([]v1.MachineType, error) {
	zoneList, err := g.ListZones()
	if err != nil {
		return nil, err
	}
	//machinesZone to keep zone to corresponding machine
	machinesZone := map[string][]string{}
	var machineTypes []v1.MachineType
	for _, zone := range zoneList {
		req := g.ComputeService.MachineTypes.List(g.GceProjectID, zone)
		err := req.Pages(g.Ctx, func(list *compute.MachineTypeList) error {
			for _, machine := range list.Items {
				res, err := ParseMachine(machine)
				if err != nil {
					return err
				}
				// to check whether we added this machine to machineTypes
				// if we found it then add this zone to machinesZone, else add the machine to machineTypes and also add this zone to machinesZone
				if zones, found := machinesZone[res.Spec.SKU]; found {
					machinesZone[res.Spec.SKU] = append(zones, zone)
				} else {
					machinesZone[res.Spec.SKU] = []string{zone}
					machineTypes = append(machineTypes, *res)
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	//update g.Data.MachineTypes[].Zones
	for index, instanceType := range machineTypes {
		machineTypes[index].Spec.Zones = machinesZone[instanceType.Spec.SKU]
	}
	return machineTypes, nil
}

func getComputeService(ctx context.Context, sajson string) (*compute.Service, error) {
	conf, err := google.JWTConfigFromJSON([]byte(sajson), compute.ComputeScope)
	if err != nil {
		return nil, err
	}
	client := conf.Client(ctx)
	return compute.New(client)
}
