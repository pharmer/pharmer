package util

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"sort"

	"github.com/pharmer/pharmer/data"
)

func CreateDir(dir string) error {
	err := os.MkdirAll(dir, 0777)
	if err != nil {
		return fmt.Errorf("failed to create dir `%s`. Reason: %v", dir, err)
	}
	return nil
}

func ReadFile(name string) ([]byte, error) {
	dataBytes, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("failed to read `%s`.Reason: %v", name, err)
	}
	return dataBytes, nil
}

func WriteFile(filename string, bytes []byte) error {
	err := ioutil.WriteFile(filename, bytes, 0666)
	if err != nil {
		return fmt.Errorf("failed to write `%s`. Reason: %v", filename, err)
	}
	return nil
}

// versions string formate is `1.1.0,1.9.0`
//they are comma seperated, no space allowed
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

// wanted directory is [path]/pharmer/data/files
// Current directory is [path]/pharmer/hack/gendata
func GetWriteDir() (string, error) {
	AbsPath, err := filepath.Abs("")
	if err != nil {
		return "", err
	}
	p := strings.TrimSuffix(AbsPath, "/hack/gendata")
	p = filepath.Join(p, "data", "files")
	return p, nil
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
	for index, _ := range data.Regions {
		sort.Slice(data.Regions[index].Zones, func(i, j int) bool {
			return data.Regions[index].Zones[i] < data.Regions[index].Zones[j]
		})
	}
	sort.Slice(data.InstanceTypes, func(i, j int) bool {
		return data.InstanceTypes[i].SKU < data.InstanceTypes[j].SKU
	})
	for index, _ := range data.InstanceTypes {
		sort.Slice(data.InstanceTypes[index].Zones, func(i, j int) bool {
			return data.InstanceTypes[index].Zones[i] < data.InstanceTypes[index].Zones[j]
		})
	}
	return data
}
