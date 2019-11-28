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
package gce

import (
	"fmt"
	"strings"

	"pharmer.dev/cloud/apis"
	v1 "pharmer.dev/cloud/apis/cloud/v1"
	"pharmer.dev/cloud/pkg/util"

	"github.com/pkg/errors"
	"google.golang.org/api/compute/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ParseRegion(region *compute.Region) (*v1.Region, error) {
	r := &v1.Region{
		Region: region.Name,
	}
	r.Zones = []string{}
	for _, url := range region.Zones {
		zone, err := ParseZoneFromUrl(url)
		if err != nil {
			return nil, err
		}
		r.Zones = append(r.Zones, zone)
	}
	return r, nil
}

func ParseZoneFromUrl(url string) (string, error) {
	words := strings.Split(url, "/")
	if len(words) == 0 {
		return "", errors.Errorf("Invaild url: unable to parse zone from url")
	}
	return words[len(words)-1], nil
}

func ParseMachine(machine *compute.MachineType) (*v1.MachineType, error) {
	return &v1.MachineType{
		ObjectMeta: metav1.ObjectMeta{
			Name: util.Sanitize(apis.GCE + "-" + machine.Name),
			Labels: map[string]string{
				apis.KeyCloudProvider: apis.GCE,
			},
		},
		Spec: v1.MachineTypeSpec{
			SKU:         machine.Name,
			Description: machine.Description,
			CPU:         resource.NewQuantity(machine.GuestCpus, resource.DecimalExponent),
			RAM:         util.QuantityP(resource.MustParse(fmt.Sprintf("%dM", machine.MemoryMb))),
			Disk:        util.QuantityP(resource.MustParse(fmt.Sprintf("%dG", machine.MaximumPersistentDisksSizeGb))),
			Category:    ParseCategoryFromSKU(machine.Name),
		},
	}, nil
}

//gce SKU format: [something]-category-[somethin/empty]
func ParseCategoryFromSKU(sku string) string {
	words := strings.Split(sku, "-")
	if len(words) < 2 {
		return ""
	} else {
		return words[1]
	}
}
