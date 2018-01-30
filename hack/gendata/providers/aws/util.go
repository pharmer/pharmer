package aws

import (
	"fmt"

	"github.com/appscode/go/log"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/pharmer/pharmer/data"
)

func ParseInstance(in *Ec2Instance) (*data.InstanceType, error) {
	out := &data.InstanceType{
		SKU:         in.Instance_type,
		Description: in.Instance_type,
		Category:    in.Family,
	}
	cpu, err := in.VCPU.Int64()
	if err != nil {
		log.Warning("ParseInstance failed, intance ", in.Instance_type, ". Reason: ", err)
		cpu = -1
	}
	out.CPU = int(cpu)
	out.RAM, err = in.Memory.Float64()
	if err != nil {
		return nil, fmt.Errorf("ParseInstance failed, intance %v. Reason: %v.", in.Instance_type, err)
	}
	return out, nil
}

func ParseRegion(in *ec2.Region) *data.Region {
	return &data.Region{
		Region: *in.RegionName,
	}
}
