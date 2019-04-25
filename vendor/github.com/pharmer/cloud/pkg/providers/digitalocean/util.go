package digitalocean

import (
	"fmt"
	"strings"

	"github.com/digitalocean/godo"
	"github.com/pharmer/cloud/pkg/apis"
	v1 "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	"github.com/pharmer/cloud/pkg/util"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ParseRegion(region *godo.Region) *v1.Region {
	return &v1.Region{
		Region: region.Slug,
		Zones: []string{
			region.Slug,
		},
		Location: region.Name,
	}
}

func ParseMachineType(sz *godo.Size) (*v1.MachineType, error) {
	return &v1.MachineType{
		ObjectMeta: metav1.ObjectMeta{
			Name: util.Sanitize(apis.DigitalOcean + "-" + sz.Slug),
			Labels: map[string]string{
				"cloud.pharmer.io/provider": apis.DigitalOcean,
			},
		},
		Spec: v1.MachineTypeSpec{
			SKU:         sz.Slug,
			Description: sz.Slug,
			CPU:         resource.NewQuantity(int64(sz.Vcpus), resource.DecimalExponent),
			RAM:         util.QuantityP(resource.MustParse(fmt.Sprintf("%dM", sz.Memory))),
			Disk:        resource.NewScaledQuantity(int64(sz.Disk), 9),
			Category:    ParseCategoryFromSlug(sz.Slug),
			Zones:       sz.Regions,
			Deprecated:  !sz.Available,
		},
	}, nil
}

func ParseCategoryFromSlug(slug string) string {
	if strings.HasPrefix(slug, "m-") {
		return "High Memory"
	} else if strings.HasPrefix(slug, "c-") {
		return "High Cpu"
	} else {
		return "Standard Droplets"
	}
}
