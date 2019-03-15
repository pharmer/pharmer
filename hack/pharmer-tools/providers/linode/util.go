package linode

import (
	"github.com/linode/linodego"
	"github.com/pharmer/pharmer/data"
	"github.com/pharmer/pharmer/hack/pharmer-tools/util"
	"github.com/pkg/errors"
)

func ParseRegion(in *linodego.Region) *data.Region {
	return &data.Region{
		Location: in.Country,
		Region:   in.ID,
		Zones: []string{
			in.ID,
		},
	}
}

func ParseInstance(in *linodego.LinodeType) (*data.InstanceType, error) {
	out := &data.InstanceType{
		SKU:         in.ID,
		Description: in.Label,
		CPU:         in.VCPUs,
		Disk:        in.Disk,
	}
	var err error
	out.RAM, err = util.MBToGB(int64(in.Memory))
	if err != nil {
		return nil, errors.Errorf("Parse Instance failed. reason: %v", err)
	}
	return out, nil
}
