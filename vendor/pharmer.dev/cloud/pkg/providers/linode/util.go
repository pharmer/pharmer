/*
Copyright The Pharmer Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package linode

import (
	"fmt"

	"pharmer.dev/cloud/apis"
	v1 "pharmer.dev/cloud/apis/cloud/v1"
	"pharmer.dev/cloud/pkg/util"

	"github.com/linode/linodego"
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
