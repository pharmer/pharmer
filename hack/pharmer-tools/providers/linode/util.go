package linode

import (
	"strconv"
	"github.com/pharmer/pharmer/data"
	"github.com/pharmer/pharmer/hack/pharmer-tools/util"
	"github.com/pkg/errors"
	"github.com/taoh/linodego"
)

func ParseRegion(in *linodego.DataCenter) *data.Region {
	return &data.Region{
		Location: in.Location,
		Region:   strconv.Itoa(in.DataCenterId),
		Zones: []string{
			strconv.Itoa(in.DataCenterId),
		},
	}
}

func ParseInstance(in *linodego.LinodePlan) (*data.InstanceType, error) {
	out := &data.InstanceType{
		SKU:         strconv.Itoa(in.PlanId),
		Description: in.Label.String(),
		CPU:         in.Cores,
		Disk:        in.Disk,
	}
	var err error
	out.RAM, err = util.MBToGB(int64(in.RAM))
	if err != nil {
		return nil, errors.Errorf("Parse Instance failed. reason: %v", err)
	}
	return out, nil
}
