package cloud

import (
	"io/ioutil"
	"os/exec"

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

func Filter(list []string, strToFilter string) (newList []string) {
	for _, item := range list {
		if item != strToFilter {
			newList = append(newList, item)
		}
	}
	return
}

func CopyDirectory(src, dst string) error {
	return exec.Command("cp", "-rf", src, dst).Run()
}
