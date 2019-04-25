package util

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/pharmer/cloud/pkg/apis"

	v1 "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/version"
	"sigs.k8s.io/yaml"
)

func QuantityP(q resource.Quantity) *resource.Quantity {
	return &q
}

func Sanitize(s string) string {
	return strings.Replace(strings.ToLower(strings.TrimSpace(s)), "_", "-", -1)
}

func ReadFile(name string) ([]byte, error) {
	dataBytes, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, err
	}
	return dataBytes, nil
}

func WriteFile(filename string, bytes []byte) error {
	err := ioutil.WriteFile(filename, bytes, 0666)
	if err != nil {
		return errors.Errorf("failed to write `%s`. Reason: %v", filename, err)
	}
	return nil
}

func MBToGB(in int64) (float64, error) {
	gb, err := strconv.ParseFloat(strconv.FormatFloat(float64(in)/1024, 'f', 2, 64), 64)
	return gb, err
}

//getting provider data from cloud.yaml file
//data contained in [path to pharmer]/data/files/[provider]/cloud.yaml
func GetDataFormFile(provider string) (*v1.CloudProvider, error) {
	data := v1.CloudProvider{}
	dir := filepath.Join(apis.DataDir, provider, "cloud.yaml")
	dataBytes, err := ReadFile(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return &v1.CloudProvider{}, nil
		}
		return nil, err
	}
	err = yaml.UnmarshalStrict(dataBytes, &data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func SortCloudProvider(data *v1.CloudProvider) *v1.CloudProvider {
	sort.Slice(data.Spec.Regions, func(i, j int) bool {
		return data.Spec.Regions[i].Region < data.Spec.Regions[j].Region
	})
	for index := range data.Spec.Regions {
		sort.Slice(data.Spec.Regions[index].Zones, func(i, j int) bool {
			return data.Spec.Regions[index].Zones[i] < data.Spec.Regions[index].Zones[j]
		})
	}
	sort.Slice(data.Spec.MachineTypes, func(i, j int) bool {
		return data.Spec.MachineTypes[i].Spec.SKU < data.Spec.MachineTypes[j].Spec.SKU
	})
	for index := range data.Spec.MachineTypes {
		sort.Slice(data.Spec.MachineTypes[index].Spec.Zones, func(i, j int) bool {
			return data.Spec.MachineTypes[index].Spec.Zones[i] < data.Spec.MachineTypes[index].Spec.Zones[j]
		})
	}
	sort.Slice(data.Spec.KubernetesVersions, func(i, j int) bool {
		return version.CompareKubeAwareVersionStrings(
			data.Spec.KubernetesVersions[i].Spec.GitVersion, data.Spec.KubernetesVersions[j].Spec.GitVersion) < 0
	})
	return data
}
