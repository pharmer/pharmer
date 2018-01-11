package providers

import (
	"github.com/pharmer/pharmer/hack/gendata/providers/gce"
	"github.com/pharmer/pharmer/data"
)

type Interface interface {
	Gce(gecProjectName,credentialFilePath string) (CloudInterface, error)
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

func (p *CloudProvider) Gce(gecProjectName,credentialFilePath string) (CloudInterface, error){
	gceClient, err := gce.NewGceClient(gecProjectName,credentialFilePath)
	if err!=nil {
		return nil, err
	}
	return gceClient, nil
}

func NewCloudProvider() *CloudProvider {
	return &CloudProvider{}
}