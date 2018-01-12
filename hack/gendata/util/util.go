package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

func CreateDir(dir string) error {
	err := os.MkdirAll(dir, 0777)
	if err != nil {
		return fmt.Errorf("failed to create dir `%s`. Reason: %v", dir, err)
	}
	return nil
}

func ReadFile(name string) ([]byte, error) {
	crtBytes, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("failed to read `%s`.Reason: %v", name, err)
	}
	return crtBytes, nil
}

func WriteFile(filename string, bytes []byte) error {
	err := ioutil.WriteFile(filename, bytes, 0666)
	if err != nil {
		return fmt.Errorf("failed to write `%s`. Reason: %v", filename, err)
	}
	return nil
}

// versions string formate is `1.1.0,1.9.0`
//they are comma seperated, no space allowed
func ParseVersions(versions string) []string {
	v := strings.Split(versions,",")
	return v
}