package cmds

import (
	"encoding/json"
	"path/filepath"

	"github.com/pharmer/pharmer/data"
	"github.com/pharmer/pharmer/hack/gendata/util"
	"github.com/pharmer/pharmer/hack/gendata/providers"
)

const (
	WriteDir string = "data"
)

func WriteCloudData(cloudInterface providers.CloudInterface) error {
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
