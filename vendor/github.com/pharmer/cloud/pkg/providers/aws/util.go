package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/pharmer/cloud/pkg/apis"
	v1 "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	"github.com/pharmer/cloud/pkg/util"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ParseInstance(in *Ec2Instance) (*v1.MachineType, error) {
	out := &v1.MachineType{
		ObjectMeta: metav1.ObjectMeta{
			Name: util.Sanitize(apis.AWS + "-" + in.InstanceType),
			Labels: map[string]string{
				"cloud.pharmer.io/provider": apis.AWS,
			},
		},
		Spec: v1.MachineTypeSpec{
			SKU:         in.InstanceType,
			Description: in.InstanceType,
			Category:    in.Family,
			CPU:         util.QuantityP(resource.MustParse(in.VCPU.String())),
			RAM:         util.QuantityP(resource.MustParse(in.Memory.String() + "Gi")),
		},
	}
	if in.Storage != nil {
		out.Spec.Disk = util.QuantityP(resource.MustParse(fmt.Sprintf("%dG", in.Storage.Size)))
	}
	return out, nil
}

func ParseRegion(in *ec2.Region) *v1.Region {
	return &v1.Region{
		Region: *in.RegionName,
	}
}
