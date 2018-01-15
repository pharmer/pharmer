package providers

import (
	"github.com/pharmer/pharmer/hack/gendata/providers/gce"
	"github.com/pharmer/pharmer/data"
	"path/filepath"
	"github.com/pharmer/pharmer/hack/gendata/util"
	"encoding/json"
	"github.com/pharmer/pharmer/hack/gendata/cmds/options"
	"fmt"
	"strings"
	"github.com/pharmer/pharmer/hack/gendata/providers/digitalocean"
)

type CloudInterface interface {
	GetName() string
	GetEnvs() []string
	GetCredentials() []data.CredentialFormat
	GetKubernets() []data.Kubernetes
	GetRegions() ([]data.Region, error)
	GetZones() ([]string, error)
	GetInstanceTypes() ([]data.InstanceType, error)
}

func NewCloudProvider(opts *options.CloudData) (CloudInterface, error) {
	switch opts.Provider {
	case "gce":
		return gce.NewGceClient(opts.GCEProjectName, opts.CredentialFile,opts.KubernetesVersions)
		break
	case "digitalocean":
		return digitalocean.NewDigitalOceanClient(opts.DoToken, opts.KubernetesVersions)
		break
	default:
		return nil, fmt.Errorf("Valid/Supported provider name required")
	}
	return nil,nil
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
	dir, err := GetWriteDir()
	if err != nil {
		return err
	}
	err = util.WriteFile(filepath.Join(dir, cloudData.Name, "cloud.json"), dataBytes)
	return err
}

// wanted directory is [path]/pharmer/data/files
// Current directory is [path]/pharmer/hack/gendata
func GetWriteDir() (string, error) {
	AbsPath, err := filepath.Abs("")
	if err!=nil {
		return "",err
	}
	p := strings.TrimSuffix(AbsPath, "/hack/gendata")
	p = filepath.Join(p,"data","files")
	return p,nil
}