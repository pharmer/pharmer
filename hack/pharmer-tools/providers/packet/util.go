package packet

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/packethost/packngo"
	"github.com/pharmer/pharmer/data"
	"github.com/pharmer/pharmer/hack/pharmer-tools/util"
)

var (
	NumOfCore = map[string]int{
		"Intel Atom C2550 @ 2.4Ghz":     4,
		"Intel E3-1240 v3":              4,
		"Intel Xeon E3-1578L v5":        4,
		"Intel Xeon E5-2650 v4 @2.2GHz": 24,
		"Cavium ThunderX CN8890 @2GHz":  96,
		"Intel E5-2640 v3":              16,
		"Intel Xeon D-1537 @1.7GHz":     16,
	}
)

func ParseFacility(facility *packngo.Facility) *data.Region {
	return &data.Region{
		Region:   facility.Name,
		Location: facility.Name,
		Zones: []string{
			facility.Code,
		},
	}
}

func ParsePlan(plan *PlanExtended) (*data.InstanceType, error) {
	ins := &data.InstanceType{
		SKU:         plan.Slug,
		Description: plan.Description,
	}
	var err error
	ins.RAM, err = RemoveUnitRetFloat64(plan.Specs.Memory.Total)
	if err != nil {
		return nil, err
	}
	ins.Disk, err = RemoveUnitRetInt(plan.Specs.Drives[0].Size)
	if err != nil {
		return nil, err
	}
	ins.CPU, err = GetCpuCore(plan.Specs.Cpus[0].Type)
	if err != nil {
		return nil, err
	}
	return ins, nil
}

//formate: "/facilities/[id]"
func GetFacilityIdFromHerf(herf string) string {
	w := strings.Split(herf, "/")
	return w[len(w)-1]
}

func GetCpuCore(name string) (int, error) {
	if core, found := NumOfCore[name]; found {
		return core, nil
	} else {
		return 0, fmt.Errorf("Can't find number of core for %v.", name)
	}
}

// 4GB -> 4
// 2048MB -> 2048/1024 -> 2
func RemoveUnitRetFloat64(in string) (float64, error) {
	if in[len(in)-2:] == "GB" {
		return strconv.ParseFloat(in[:len(in)-2], 64)
	} else if in[len(in)-2:] == "MB" {
		val, err := strconv.ParseInt(in[:len(in)-2], 10, 64)
		if err != nil {
			return 0, err
		}
		return util.MBToGB(val)
	} else {
		return 0, fmt.Errorf("Invalid unit: %v.", in)
	}
}
func RemoveUnitRetInt(in string) (int, error) {
	val, err := RemoveUnitRetFloat64(in)
	return int(val), err
}
