package vultr

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/pharmer/pharmer/hack/pharmer-tools/util"
)

func TestRegion(t *testing.T) {
	client, err := NewVultrClient(tgetToken(), "1.1.2")
	if err != nil {
		t.Error(err)
	}
	regions, err := client.GetRegions()
	if err != nil {
		t.Error(err)
	}
	fmt.Println(regions)
}

func TestInstance(t *testing.T) {
	client, err := NewVultrClient(tgetToken(), "1.1.2")
	if err != nil {
		t.Error(err)
	}
	instances, err := client.GetInstanceTypes()
	if err != nil {
		t.Error(err)
	}
	for _, i := range instances {
		fmt.Println(i.SKU)
	}
	fmt.Println("total:", len(instances))
}

func tgetToken() string {
	b, _ := util.ReadFile("/home/ac/Downloads/cred/vultr.json")
	v := struct {
		Token string `json:"token"`
	}{}
	fmt.Println(json.Unmarshal(b, &v))
	//fmt.Println(v)
	return v.Token
}
