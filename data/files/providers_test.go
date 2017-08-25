package files_test

import (
	"fmt"
	"log"
	"testing"

	_env "github.com/appscode/go/env"
	. "github.com/appscode/pharmer/data/files"
)

func TestName(t *testing.T) {
	GetInstanceType("digitalocean", "do.8gb")
	GetInstanceType("gce", "n1-standard-2")
}

func TestLoadKubernetesVersion(t *testing.T) {
	provider := "aws"
	k, err := GetClusterVersion(provider, "qa", "1.4.4")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Cluster (%v, %v, %v) is deprecated? %v\n", provider, "qa", "1.4.4", k.Deprecated)
	k, err = GetClusterVersion(provider, "prod", "1.4.5")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Cluster (%v, %v, %v) is deprecated? %v\n", provider, "qa", "1.4.5", k.Deprecated)

	provider = "gce"
	k, err = GetClusterVersion(provider, "qa", "1.4.4")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Cluster (%v, %v, %v) is deprecated? %v\n", provider, "qa", "1.4.4", k.Deprecated)
	k, err = GetClusterVersion(provider, "prod", "1.4.5")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Cluster (%v, %v, %v) is deprecated? %v\n", provider, "qa", "1.4.5", k.Deprecated)

	provider = "digitalocean"
	k, err = GetClusterVersion(provider, "qa", "1.4.4")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Cluster (%v, %v, %v) is deprecated? %v\n", provider, "qa", "1.4.4", k.Deprecated)

	provider = "linode"
	k, err = GetClusterVersion(provider, "qa", "1.4.4")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Cluster (%v, %v, %v) is deprecated? %v\n", provider, "qa", "1.4.4", k.Deprecated)

	provider = "vultr"
	k, err = GetClusterVersion(provider, "qa", "1.4.4")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Cluster (%v, %v, %v) is deprecated? %v\n", provider, "qa", "1.4.4", k.Deprecated)
}

func TestLoadForProdEnv(t *testing.T) {
	if err := Load(_env.Prod); err != nil {
		t.Fatal(err)
	}
}

func TestLoadForQAEnv(t *testing.T) {
	if err := Load(_env.QA); err != nil {
		t.Fatal(err)
	}
}

func TestLoadForDevEnv(t *testing.T) {
	if err := Load(_env.Dev); err != nil {
		t.Fatal(err)
	}
}
