package credential

import (
	"encoding/json"
	"io/ioutil"
)

type GCE struct {
	CommonSpec
}

func NewGCE() *GCE {
	return &GCE{}
}

func (c GCE) ProjectID() string      { return c.Data[GCEProjectID] }
func (c GCE) ServiceAccount() string { return c.Data[GCEServiceAccount] }

func (c *GCE) Load(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	d := make(map[string]string)
	if err = json.Unmarshal(data, &d); err != nil {
		return err
	}

	c.Data = make(map[string]string)
	c.Data[GCEServiceAccount] = string(data)
	c.Data[GCEProjectID] = d["project_id"]
	return nil
}
