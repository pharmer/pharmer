package data_test

import (
	"encoding/json"
	"fmt"
	"log"
	"testing"

	. "github.com/appscode/data"
	"github.com/appscode/data/files"
	"github.com/stretchr/testify/assert"
)

func TestGenericParsing(t *testing.T) {
	bytes, err := files.Asset("files/pkg.latest.json")
	if err != nil {
		log.Fatal(err)
	}

	p := &struct {
		Product []*Product `json:"pkg"`
	}{}
	err = json.Unmarshal(bytes, &p)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(p.Product[0].DisplayName)
	fmt.Println(string(p.Product[0].Metadata))
}

func TestPkgData(t *testing.T) {
	pkg, err := LoadLatestPkgData()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(pkg.Package[0].DisplayName)
	fmt.Println(string(pkg.Package[0].Metadata))
}

func TestName(t *testing.T) {
	ClusterMachineType("digitalocean", "do.8gb")
	ClusterMachineType("gce", "n1-standard-2")
}

func TestLoadKubernetesVersion(t *testing.T) {
	provider := "aws"
	k, err := LoadKubernetesVersion(provider, "qa", "1.4.4")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Cluster (%v, %v, %v) is deprecated? %v\n", provider, "qa", "1.4.4", k.Deprecated)
	k, err = LoadKubernetesVersion(provider, "prod", "1.4.5")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Cluster (%v, %v, %v) is deprecated? %v\n", provider, "qa", "1.4.5", k.Deprecated)

	provider = "gce"
	k, err = LoadKubernetesVersion(provider, "qa", "1.4.4")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Cluster (%v, %v, %v) is deprecated? %v\n", provider, "qa", "1.4.4", k.Deprecated)
	k, err = LoadKubernetesVersion(provider, "prod", "1.4.5")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Cluster (%v, %v, %v) is deprecated? %v\n", provider, "qa", "1.4.5", k.Deprecated)

	provider = "digitalocean"
	k, err = LoadKubernetesVersion(provider, "qa", "1.4.4")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Cluster (%v, %v, %v) is deprecated? %v\n", provider, "qa", "1.4.4", k.Deprecated)

	provider = "linode"
	k, err = LoadKubernetesVersion(provider, "qa", "1.4.4")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Cluster (%v, %v, %v) is deprecated? %v\n", provider, "qa", "1.4.4", k.Deprecated)

	provider = "vultr"
	k, err = LoadKubernetesVersion(provider, "qa", "1.4.4")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Cluster (%v, %v, %v) is deprecated? %v\n", provider, "qa", "1.4.4", k.Deprecated)
}

func TestKubernetesPricingMetadata(t *testing.T) {
	tx, err := KubernetesPricingMetadata("gce", "n1-standard-2")
	assert.Nil(t, err)
	fmt.Println(tx)

	tx2, err := KubernetesPricingMetadata("aws", "t2.micro")
	assert.Nil(t, err)
	fmt.Println(tx2)
}

func TestDNSProviderData(t *testing.T) {
	dns, err := LoadDNSProviderData()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(dns[0].Fields[0].Name)
	//fmt.Println(ic.Command[0].Description)
}
