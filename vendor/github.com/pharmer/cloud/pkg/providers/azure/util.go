package azure

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/profiles/2019-03-01/resources/mgmt/subscriptions"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"github.com/pharmer/cloud/pkg/apis"
	v1 "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	"github.com/pharmer/cloud/pkg/util"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ParseRegion(in *subscriptions.Location) *v1.Region {
	return &v1.Region{
		Region: *in.DisplayName,
		Zones: []string{
			*in.Name,
		},
	}
}

func ParseInstance(in *compute.VirtualMachineSize) (*v1.MachineType, error) {
	out := &v1.MachineType{
		ObjectMeta: metav1.ObjectMeta{
			Name: util.Sanitize(apis.Azure + "-" + *in.Name),
			Labels: map[string]string{
				apis.KeyCloudProvider: apis.Azure,
			},
		},
		Spec: v1.MachineTypeSpec{
			SKU:         *in.Name,
			Description: *in.Name,
			CPU:         resource.NewQuantity(int64(*in.NumberOfCores), resource.DecimalExponent),
			RAM:         util.QuantityP(resource.MustParse(fmt.Sprintf("%dM", *in.MemoryInMB))),
			Disk:        util.QuantityP(resource.MustParse(fmt.Sprintf("%dM", *in.OsDiskSizeInMB))),
		},
	}
	return out, nil
}
