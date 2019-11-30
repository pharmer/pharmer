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
package aws

import (
	"fmt"

	"pharmer.dev/cloud/apis"
	v1 "pharmer.dev/cloud/apis/cloud/v1"
	"pharmer.dev/cloud/pkg/util"

	"github.com/aws/aws-sdk-go/service/ec2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ParseInstance(in *Ec2Instance) (*v1.MachineType, error) {
	out := &v1.MachineType{
		ObjectMeta: metav1.ObjectMeta{
			Name: util.Sanitize(apis.AWS + "-" + in.InstanceType),
			Labels: map[string]string{
				apis.KeyCloudProvider: apis.AWS,
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
