package gce

import (
	"google.golang.org/api/compute/v1"
	"github.com/pharmer/pharmer/data"
	"strings"
	"fmt"
	"strconv"
)

const (
	CatagoryUnknown string = "unknown"
)

func ParseRegion(region *compute.Region) (*data.Region, error) {
	r := &data.Region{
		Region: region.Name,
	}
	r.Zones = []string{}
	for _,url := range region.Zones {
		zone, err := ParseZoneFromUrl(url)
		if err!=nil {
			return nil, err
		}
		r.Zones = append(r.Zones, zone)
	}

	return r, nil
}

func ParseZoneFromUrl(url string) (string, error){
	words := strings.Split(url, "/")
	if len(words)==0 {
		return "",fmt.Errorf("Invaild url: unable to parse zone from url")
	}
	return words[len(words)-1],nil
}

func ParseMachine(machine *compute.MachineType) (*data.InstanceType, error) {
	m := &data.InstanceType{
		SKU: machine.Name,
		Description: machine.Description,
		CPU: int(machine.GuestCpus),
		Disk: int(machine.MaximumPersistentDisks),
		Category: ParseCatagoryFromSKU(machine.Name),
	}

	var err error
	m.RAM, err =strconv.ParseFloat(strconv.FormatFloat(float64(machine.MemoryMb)/1024,'f',2,64), 64)
	return m,err
}

//gce SKU formate: [something]-catagory-[somethin/empty]
func ParseCatagoryFromSKU(sku string) string {
	words := strings.Split(sku,"-")
	if len(words)<2 {
		return CatagoryUnknown
	} else {
		return words[1]
	}
}