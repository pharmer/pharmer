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
package azure

import (
	"fmt"

	"pharmer.dev/cloud/apis"
	v1 "pharmer.dev/cloud/apis/cloud/v1"
	"pharmer.dev/cloud/pkg/util"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-06-01/subscriptions"
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
