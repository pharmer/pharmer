package linode

import (
	"fmt"

	"github.com/linode/linodego"
	"github.com/pharmer/cloud/pkg/apis"
	v1 "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	"github.com/pharmer/cloud/pkg/util"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ParseRegion(in *linodego.Region) *v1.Region {
	return &v1.Region{
		Location: in.Country,
		Region:   in.ID,
		Zones: []string{
			in.ID,
		},
	}
}

func ParseInstance(in *linodego.LinodeType) (*v1.MachineType, error) {
	return &v1.MachineType{
		ObjectMeta: metav1.ObjectMeta{
			Name: util.Sanitize(apis.Linode + "-" + in.ID),
			Labels: map[string]string{
				apis.KeyCloudProvider: apis.Linode,
			},
		},
		Spec: v1.MachineTypeSpec{
			SKU:         in.ID,
			Description: in.Label,
			CPU:         resource.NewQuantity(int64(in.VCPUs), resource.DecimalExponent),
			RAM:         util.QuantityP(resource.MustParse(fmt.Sprintf("%dMi", in.Memory))),
			Disk:        util.QuantityP(resource.MustParse(fmt.Sprintf("%dMi", in.Disk))),
		},
	}, nil
}
