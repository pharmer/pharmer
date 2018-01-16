package gce

import (
	"context"

	"github.com/pharmer/pharmer/credential"
	"github.com/pharmer/pharmer/data"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
)

type GceClient struct {
	Data *GceDefaultData `json:"data,omitempty"`
	GceProjectName     string           `json:"gceProjectName,omitempty"`
	ComputeService     *compute.Service `json:"compute_service,omitempty"`
	Ctx                context.Context  `json:"ctx,omitempty"`
}

type GceDefaultData struct {
	Name        string                       `json:"name"`
	Envs        []string                     `json:"envs,omitempty"`
	Credentials []data.CredentialFormat `json:"credentials"`
	Kubernetes  []data.Kubernetes       `json:"kubernetes"`
}

func NewGceClient(gecProjectName, credentialFilePath, versions string) (*GceClient, error) {
	g := &GceClient{
		GceProjectName:     gecProjectName,
		Ctx:                context.Background(),
		Data:               &GceDefaultData{},
	}
	var err error
	g.ComputeService, err = getComputeService(g.Ctx, credentialFilePath)
	if err != nil {
		return nil, err
	}
	g.Data, err = GetDefault(versions)
	if err != nil {
		return nil, err
	}
	return g, nil
}

func (g *GceClient) GetName() string {
	return g.Data.Name
}

func (g *GceClient) GetEnvs() []string {
	return g.Data.Envs
}

func (g *GceClient) GetCredentials() []data.CredentialFormat {
	return g.Data.Credentials
}

func (g *GceClient) GetKubernets() []data.Kubernetes {
	return g.Data.Kubernetes
}

func (g *GceClient) GetRegions() ([]data.Region, error) {
	req := g.ComputeService.Regions.List(g.GceProjectName)

	regions := []data.Region{}
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

func (g *GceClient) GetZones() ([]string, error) {
	req := g.ComputeService.Zones.List(g.GceProjectName)
	zones := []string{}
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

func (g *GceClient) GetInstanceTypes() ([]data.InstanceType, error) {
	zoneList, err := g.GetZones()
	if err != nil {
		return nil, err
	}
	//machinesZone to keep zone to corresponding machine
	machinesZone := map[string][]string{}
	instanceTypes := []data.InstanceType{}
	for _, zone := range zoneList {
		req := g.ComputeService.MachineTypes.List(g.GceProjectName, zone)
		err := req.Pages(g.Ctx, func(list *compute.MachineTypeList) error {
			for _, machine := range list.Items {
				res, err := ParseMachine(machine)
				if err != nil {
					return err
				}
				// to check whether we added this machine to instanceTypes
				// if we found it then add this zone to machinesZone, else add the machine to instanceTypes and also add this zone to machinesZone
				if zones, found := machinesZone[res.SKU]; found {
					machinesZone[res.SKU] = append(zones, zone)
				} else {
					machinesZone[res.SKU] = []string{zone}
					instanceTypes = append(instanceTypes, *res)
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	//update g.Data.InstanceTypes[].Zones
	for index, instanceType := range instanceTypes {
		instanceTypes[index].Zones = machinesZone[instanceType.SKU]
	}
	return instanceTypes, nil
}

func getComputeService(ctx context.Context, credentialFilePath string) (*compute.Service, error) {
	gceInfo := credential.NewGCE()
	err := gceInfo.Load(credentialFilePath)
	if err != nil {
		return nil, err
	}
	conf, err := google.JWTConfigFromJSON([]byte(gceInfo.ServiceAccount()),
		compute.ComputeScope)
	if err != nil {
		return nil, err
	}
	client := conf.Client(ctx)
	return compute.New(client)
}
