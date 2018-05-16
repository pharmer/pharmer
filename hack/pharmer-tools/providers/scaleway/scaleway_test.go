package scaleway

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/pharmer/pharmer/hack/pharmer-tools/util"
)

func TestInstance(t *testing.T) {
	client, err := NewClient(tgetToken(), tgetOrganization(), "1.1.1")
	if err != nil {
		t.Error(err)
	}
	insList, err := client.GetInstanceTypes()
	if err != nil {
		t.Error(err)
	}
	fmt.Println(insList)
}

func tgetToken() string {
	b, _ := util.ReadFile("/home/ac/Downloads/cred/scaleway.json")
	v := struct {
		Token        string `json:"token"`
		Organization string `json:"organization"`
	}{}
	fmt.Println(json.Unmarshal(b, &v))
	//fmt.Println(v)
	return v.Token
}

func tgetOrganization() string {
	b, _ := util.ReadFile("/home/ac/Downloads/cred/scaleway.json")
	v := struct {
		Token        string `json:"token"`
		Organization string `json:"organization"`
	}{}
	fmt.Println(json.Unmarshal(b, &v))
	//fmt.Println(v)
	return v.Organization
}
