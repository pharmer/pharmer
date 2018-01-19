package vultr

import (
	"fmt"
	"strconv"

	vultr "github.com/JamesClonk/vultr/lib"
	"github.com/pharmer/pharmer/data"
	"github.com/pharmer/pharmer/hack/gendata/util"
)

func ParseRegion(in *vultr.Region) *data.Region {
	return &data.Region{
		Location: in.Name,
		Region:   strconv.Itoa(in.ID),
		Zones: []string{
			strconv.Itoa(in.ID),
		},
	}
}

func ParseInstance(in *PlanExtended) (*data.InstanceType, error) {
	out := &data.InstanceType{
		SKU:         strconv.Itoa(in.ID),
		Description: in.Name,
		CPU:         in.VCpus,
		Category:    in.Catagory,
	}
	var err error
	disk, err := strconv.ParseInt(in.Disk, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("Parse Instance failed.reasion: %v.", err)
	}
	out.Disk = int(disk)
	ram, err := strconv.ParseInt(in.RAM, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("Parse Instance failed.reasion: %v.", err)
	}
	out.RAM, err = util.MBToGB(ram)
	if err != nil {
		return nil, fmt.Errorf("Parse Instance failed.reasion: %v.", err)
	}
	out.Regions = []string{}
	for _, r := range in.Regions {
		region := strconv.Itoa(r)
		out.Regions = append(out.Regions, region)
	}
	return out, nil
}
