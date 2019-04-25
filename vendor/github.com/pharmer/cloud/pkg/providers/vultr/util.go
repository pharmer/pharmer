package vultr

import (
	"strconv"

	vultr "github.com/JamesClonk/vultr/lib"
	"github.com/pharmer/cloud/pkg/apis"
	v1 "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	"github.com/pharmer/cloud/pkg/util"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ParseRegion(in *vultr.Region) *v1.Region {
	return &v1.Region{
		Location: in.Name,
		Region:   strconv.Itoa(in.ID),
		Zones: []string{
			strconv.Itoa(in.ID),
		},
	}
}

func ParseInstance(in *PlanExtended) (*v1.MachineType, error) {
	out := &v1.MachineType{
		ObjectMeta: metav1.ObjectMeta{
			Name: util.Sanitize(apis.Vultr + "-" + strconv.Itoa(in.ID)),
			Labels: map[string]string{
				"cloud.pharmer.io/provider": apis.Vultr,
			},
		},
		Spec: v1.MachineTypeSpec{
			SKU:         strconv.Itoa(in.ID),
			Description: in.Name,
			CPU:         resource.NewQuantity(int64(in.VCpus), resource.DecimalExponent),
			Category:    in.Category,
		},
	}
	if in.Deprecated {
		out.Spec.Deprecated = in.Deprecated
	}

	disk, err := resource.ParseQuantity(in.Disk + "G")
	if err != nil {
		return nil, errors.Errorf("Parse Instance failed.reason: %v.", err)
	}
	out.Spec.Disk = &disk

	ram, err := resource.ParseQuantity(in.RAM + "M")
	if err != nil {
		return nil, errors.Errorf("Parse Instance failed.reason: %v.", err)
	}
	out.Spec.RAM = &ram

	out.Spec.Zones = []string{}
	for _, r := range in.Regions {
		region := strconv.Itoa(r)
		out.Spec.Zones = append(out.Spec.Zones, region)
	}
	return out, nil
}
