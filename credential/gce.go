package credential

import (
	"encoding/json"
	"io/ioutil"
)

type GCE struct {
	CommonSpec
}

func (c GCE) ProjectID() string      { return c.Data[GCEProjectID] }
func (c GCE) ServiceAccount() string { return c.Data[GCEServiceAccount] }

func (c GCE) Load(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	d := map[string]string{}
	err = json.Unmarshal(data, &data)
	if err != nil {
		return err
	}

	if c.Data != nil {
		c.Data = map[string]string{}
	}
	c.Data[GCEServiceAccount] = string(data)
	c.Data[GCEProjectID] = d["project_id"]
	return nil
}
