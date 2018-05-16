package util

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/appscode/go/runtime"
	"github.com/pharmer/pharmer/data"
	"github.com/pkg/errors"
)

func CreateDir(dir string) error {
	err := os.MkdirAll(dir, 0777)
	if err != nil {
		return errors.Errorf("failed to create dir `%s`. Reason: %v", dir, err)
	}
	return nil
}

func ReadFile(name string) ([]byte, error) {
	dataBytes, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, errors.Errorf("failed to read `%s`.Reason: %v", name, err)
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

// versions string formate is `1.1.0,1.9.0`
//they are comma separated, no space allowed
func ParseVersions(versions string) []string {
	v := strings.Split(versions, ",")
	return v
}

func MBToGB(in int64) (float64, error) {
	gb, err := strconv.ParseFloat(strconv.FormatFloat(float64(in)/1024, 'f', 2, 64), 64)
	return gb, err
}

func BToGB(in int64) (float64, error) {
	gb, err := strconv.ParseFloat(strconv.FormatFloat(float64(in)/(1024*1024*1024), 'f', 2, 64), 64)
	return gb, err
}

// write directory is [path]/pharmer/data/files
func GetWriteDir() (string, error) {
	dir := filepath.Join(runtime.GOPath(), "src/github.com/pharmer/pharmer/data/files")
	return dir, nil
}

//getting provider data from cloud.json file
//data contained in [path to pharmer]/data/files/[provider]/cloud.json
func GetDataFormFile(provider string) (*data.CloudData, error) {
	data := data.CloudData{}
	dir, err := GetWriteDir()
	if err != nil {
		return nil, err
	}
	dir = filepath.Join(dir, provider, "cloud.json")
	dataBytes, err := ReadFile(dir)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(dataBytes, &data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func SortCloudData(data *data.CloudData) *data.CloudData {
	sort.Slice(data.Regions, func(i, j int) bool {
		return data.Regions[i].Region < data.Regions[j].Region
	})
	for index := range data.Regions {
		sort.Slice(data.Regions[index].Zones, func(i, j int) bool {
			return data.Regions[index].Zones[i] < data.Regions[index].Zones[j]
		})
	}
	sort.Slice(data.InstanceTypes, func(i, j int) bool {
		return data.InstanceTypes[i].SKU < data.InstanceTypes[j].SKU
	})
	for index := range data.InstanceTypes {
		sort.Slice(data.InstanceTypes[index].Zones, func(i, j int) bool {
			return data.InstanceTypes[index].Zones[i] < data.InstanceTypes[index].Zones[j]
		})
	}
	sort.Slice(data.Kubernetes, func(i, j int) bool {
		return data.Kubernetes[i].Version.LessThan(data.Kubernetes[j].Version)
	})
	return data
}
