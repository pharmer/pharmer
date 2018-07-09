package azure

import (
	"github.com/Azure/azure-sdk-for-go/profiles/2017-03-09/resources/mgmt/subscriptions"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2017-12-01/compute"
	"github.com/pharmer/pharmer/data"
	"github.com/pharmer/pharmer/hack/pharmer-tools/util"
	"github.com/pkg/errors"
)

func ParseRegion(in *subscriptions.Location) *data.Region {
	return &data.Region{
		Region: *in.DisplayName,
		Zones: []string{
			*in.Name,
		},
	}
}

func ParseInstance(in *compute.VirtualMachineSize) (*data.InstanceType, error) {
	out := &data.InstanceType{
		SKU:         *in.Name,
		Description: *in.Name,
		CPU:         int(*in.NumberOfCores),
	}
	var err error
	out.RAM, err = util.MBToGB(int64(*in.MemoryInMB))
	if err != nil {
		return nil, errors.Errorf("Failed to parse instance. reason : %v,", err)
	}
	disk, err := util.MBToGB(int64(*in.ResourceDiskSizeInMB))
	if err != nil {
		return nil, errors.Errorf("Failed to parse instance. reason : %v,", err)
	}
	out.Disk = int(disk)
	return out, nil
}
