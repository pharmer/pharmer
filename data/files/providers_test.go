package files_test

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	version "github.com/appscode/go-version"
	_env "github.com/appscode/go/env"
	. "github.com/pharmer/pharmer/data/files"
	"github.com/stretchr/testify/assert"
)

func TestName(t *testing.T) {
	GetInstanceType("digitalocean", "do.8gb")

	im, err := GetInstanceType("gce", "n1-standard-2")
	fmt.Println(im, err)
}

func TestLoadForProdEnv(t *testing.T) {
	if err := Load(_env.Prod); err != nil {
		t.Fatal(err)
	}
	reg, err := GetRegions("gce")
	fmt.Println(err)
	keys := reflect.ValueOf(reg).MapKeys()
	fmt.Println(keys)
	fmt.Println(reg["Western Europe"].Zones)

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

func TestVersion(t *testing.T) {
	v170, _ := version.NewVersion("1.7.0")
	v180, _ := version.NewVersion("1.8.0")
	v190, _ := version.NewVersion("1.9.0")
	v181, _ := version.NewVersion("1.8.1")

	c2, err := version.NewConstraint(fmt.Sprintf(">= %s", v181.Clone().ToMutator().ResetPrerelease().ResetMetadata().ResetPatch().Done().String()))
	if err != nil {
		t.Fatal(err)
	}

	versions := []*version.Version{v170, v190, v180}
	pos := sort.Search(len(versions), func(i int) bool { return c2.Check(versions[i]) })
	assert.Equal(t, 1, pos)
}

func TestInstanceZone(t *testing.T) {

	if err := Load(_env.Prod); err != nil {
		//t.Fatal(err)
	}
	fmt.Println(GetInstanceType("packet", "baremetal_1"))
	fmt.Println(GetInstanceByZoneCPU("packet", "ord1", 2))
}
