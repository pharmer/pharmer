package files_test

import (
	"testing"

	_env "github.com/appscode/go/env"
	. "github.com/appscode/pharmer/data/files"
)

func TestName(t *testing.T) {
	GetInstanceType("digitalocean", "do.8gb")
	GetInstanceType("gce", "n1-standard-2")
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
