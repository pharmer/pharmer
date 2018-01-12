package providers

import (
	"github.com/pharmer/pharmer/hack/gendata/providers/gce"
	"github.com/pharmer/pharmer/data"
	"path/filepath"
	"github.com/pharmer/pharmer/hack/gendata/util"
	"encoding/json"
)

const (
	WriteDir string = "data"
)

type Interface interface {
	Gce(gecProjectName string,credentialFilePath string, versions []string) (CloudInterface, error)
} 

type CloudInterface interface {
	GetName() string
	GetEnvs() []string
	GetCredentials() []data.CredentialFormat
	GetKubernets() []data.Kubernetes
	GetRegions() ([]data.Region, error)
	GetZones() ([]string, error)
	GetInstanceTypes() ([]data.InstanceType, error)
}

type CloudProvider struct {}

func (p *CloudProvider) Gce(gecProjectName string,credentialFilePath string, versions string) (CloudInterface, error){
	gceClient, err := gce.NewGceClient(gecProjectName,credentialFilePath,versions)
	if err!=nil {
		return nil, err
	}
	return gceClient, nil
}

func NewCloudProvider() *CloudProvider {
	return &CloudProvider{}
}

func WriteCloudData(cloudInterface CloudInterface) error {
	var err error
	cloudData := data.CloudData{
		Name:        cloudInterface.GetName(),
		Envs:        cloudInterface.GetEnvs(),
		Credentials: cloudInterface.GetCredentials(),
		Kubernetes:  cloudInterface.GetKubernets(),
	}
	cloudData.Regions, err = cloudInterface.GetRegions()
	if err != nil {
		return err
	}
	cloudData.InstanceTypes, err = cloudInterface.GetInstanceTypes()
	if err != nil {
		return err
	}
	dataBytes, err := json.MarshalIndent(cloudData, "", "  ")
	err = util.CreateDir(WriteDir)
	if err != nil {
		return err
	}
	err = util.WriteFile(filepath.Join(WriteDir, cloudData.Name, "cloud.json"), dataBytes)
	return err
}