package scaleway

import (
	scaleway "github.com/scaleway/scaleway-cli/pkg/api"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"pharmer.dev/cloud/pkg/apis"
	v1 "pharmer.dev/cloud/pkg/apis/cloud/v1"
	"pharmer.dev/cloud/pkg/util"
)

func ParseInstance(name string, in *scaleway.ProductServer) (*v1.MachineType, error) {
	out := &v1.MachineType{
		ObjectMeta: metav1.ObjectMeta{
			Name: util.Sanitize(apis.Packet + "-" + name),
			Labels: map[string]string{
				apis.KeyCloudProvider: apis.Packet,
			},
		},
		Spec: v1.MachineTypeSpec{
			SKU:         name,
			Description: in.Arch,
			CPU:         resource.NewQuantity(int64(in.Ncpus), resource.DecimalExponent),
			RAM:         resource.NewQuantity(int64(in.Ram), resource.BinarySI),
			Disk:        resource.NewQuantity(int64(in.VolumesConstraint.MinSize), resource.DecimalSI),
		},
	}
	//if in.Baremetal {
	//	out.Category = "BareMetal"
	//} else {
	//	out.Category = "Cloud Servers"
	//}
	return out, nil
}
