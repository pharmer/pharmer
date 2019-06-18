package cloud

import (
	"io/ioutil"

	"github.com/ghodss/yaml"
	"github.com/pharmer/pharmer/store"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func getLeaderMachine(machineStore store.MachineStore, clusterName string) (*clusterapi.Machine, error) {
	machine, err := machineStore.Get(clusterName + "-master-0")
	if err != nil {
		return nil, err
	}
	return machine, nil
}

func ReadFileAs(path string, obj interface{}) error {
	d, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(d, obj)
	if err != nil {
		return err
	}
	return nil
}
