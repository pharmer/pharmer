package gce

import (
	"context"
	"encoding/json"
	"fmt"

	logs "github.com/appscode/go/log"
	"github.com/pharmer/pharmer/hack/gendata/credential"
	"github.com/pharmer/pharmer/data"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
	"github.com/pharmer/pharmer/hack/gendata/util"
)

const (
	DefaultDataFile string = "providers/gce/default.json"
)

type GceClient struct {
	Data *GceDefaultData `json:"data,omitempty"`
	//path to the credential file
	CredentialFilePath string           `json:"credentialFilePath,omitempty"`
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

func NewGceClient(gecProjectName, credentialFilePath string) (*GceClient, error) {
	g := &GceClient{
		CredentialFilePath: credentialFilePath,
		GceProjectName:     gecProjectName,
		Ctx:                context.Background(),
		Data:               &GceDefaultData{},
	}
	var err error
	g.ComputeService, err = getComputeService(g.Ctx, g.CredentialFilePath)
	if err != nil {
		return nil, err
	}
	err = g.Data.defaultData()
	if err != nil {
		return nil, err
	}
	return g, nil
}

// assign default data from gendata/providers/gce/default.json
func (d *GceDefaultData) defaultData() error {
	DataBytes, err := util.ReadFile(DefaultDataFile)
	if err != nil {
		return err
	}
	err = json.Unmarshal(DataBytes, d)
	if err != nil {
		return err
	}
	logs.Debug("Default data :", *d)
	if len(d.Name) == 0 {
		return fmt.Errorf("`%s` Name not found", DefaultDataFile)
	}
	return nil
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

//Here, we use data from GceClient.Data.Regions
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

/*
func (g *GceClient) WriteData() error {
	bytes, err := json.MarshalIndent(*g.Data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to convert GceData to json for gce. Reason: %v", err)
	}
	err = providers.CreateDir(WriteDir)
	if err != nil {
		return fmt.Errorf("failed to create directory data/gec/ for gce. Reason: %v", err)
	}
	err = providers.WriteFile(filepath.Join(WriteDir, "cloud.json"), bytes)
	return err
}

func (g *GceClient) GetData() error {
	logs.Info("Initializing..")
	err := g.Init()
	if err != nil {
		return err
	}
	logs.Info("Getting default data..")
	err = g.GetDefaultData()
	if err != nil {
		return err
	}
	logs.Info("Getting regions..")
	//GetRegion must appear after init()
	//because g.ComputeService, g.Ctx needs to initialize
	err = g.GetRegions()
	if err != nil {
		return err
	}
	logs.Info("Getting instanceTypes..")
	//GetInstanceTypes() must appear after GetRegion(),
	//because it uses GetZones() which has dependency on GetRegion()
	err = g.GetInstanceTypes()
	if err != nil {
		return err
	}
	logs.Info("Writing data..")
	err = g.WriteData()
	if err != nil {
		return err
	}
	return nil
}
*/
