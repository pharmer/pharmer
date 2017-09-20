package util

import (
	"io/ioutil"

	"github.com/ghodss/yaml"
)

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
