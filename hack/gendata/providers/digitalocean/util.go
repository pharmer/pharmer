package digitalocean

import (
	"strings"

	"github.com/digitalocean/godo"
	"github.com/pharmer/pharmer/data"
	"github.com/pharmer/pharmer/hack/gendata/util"
)

func ParseRegion(region *godo.Region) *data.Region {
	return &data.Region{
		Location: region.Name,
		Region:   region.Slug,
		Zones: []string{
			region.Slug,
		},
	}
}

func ParseSizes(size *godo.Size) (*data.InstanceType, error) {
	m := &data.InstanceType{
		SKU:         size.Slug,
		Description: size.Slug,
		CPU:         size.Vcpus,
		Disk:        size.Disk,
		Category:    ParseCatagoryFromSlug(size.Slug),
		Regions:     size.Regions,
	}
	var err error
	m.RAM, err = util.MBToGB(int64(size.Memory))
	return m, err
}

func ParseCatagoryFromSlug(slug string) string {
	if strings.HasPrefix(slug, "m-") {
		return "High Memory"
	} else if strings.HasPrefix(slug, "c-") {
		return "High Cpu"
	} else {
		return "General Purpose"
	}
}
