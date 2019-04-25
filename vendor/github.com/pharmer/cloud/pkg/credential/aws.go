package credential

import (
	ini "gopkg.in/ini.v1"
)

type AWS struct {
	CommonSpec
}

func NewAWS() *AWS {
	return &AWS{}
}

func (c AWS) AccessKeyID() string     { return c.Data[AWSAccessKeyID] }
func (c AWS) SecretAccessKey() string { return c.Data[AWSSecretAccessKey] }

func (c *AWS) Load(filename string) error {
	c.Data = make(map[string]string)

	cfg, err := ini.Load(filename)
	if err != nil {
		return err
	}
	sec, err := cfg.GetSection("default")
	if err != nil {
		return err
	}

	id, err := sec.GetKey("aws_access_key_id")
	if err != nil {
		return err
	}
	c.Data[AWSAccessKeyID] = id.Value()

	secret, err := sec.GetKey("aws_secret_access_key")
	if err != nil {
		return err
	}
	c.Data[AWSSecretAccessKey] = secret.Value()

	return nil
}
