package vultr

import (
	"strconv"

	vultr "github.com/JamesClonk/vultr/lib"
	"github.com/pharmer/pharmer/data"
	"github.com/pharmer/pharmer/hack/pharmer-tools/util"
	"github.com/pkg/errors"
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
		Category:    in.Category,
	}
	if in.Deprecated {
		out.Deprecated = in.Deprecated
	}
	var err error
	disk, err := strconv.ParseInt(in.Disk, 10, 64)
	if err != nil {
		return nil, errors.Errorf("Parse Instance failed.reasion: %v.", err)
	}
	out.Disk = int(disk)
	ram, err := strconv.ParseInt(in.RAM, 10, 64)
	if err != nil {
		return nil, errors.Errorf("Parse Instance failed.reasion: %v.", err)
	}
	out.RAM, err = util.MBToGB(ram)
	if err != nil {
		return nil, errors.Errorf("Parse Instance failed.reasion: %v.", err)
	}
	out.Zones = []string{}
	for _, r := range in.Regions {
		region := strconv.Itoa(r)
		out.Zones = append(out.Zones, region)
	}
	return out, nil
}
