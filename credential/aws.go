package credential

import (
	"gopkg.in/ini.v1"
)

type AWS struct {
	generic
}

func (s AWS) Load(filename string) error {
	if s.Data != nil {
		s.Data = map[string]string{}
	}

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
	s.Data[AWSAccessKeyID] = id.Value()

	secret, err := sec.GetKey("aws_secret_access_key")
	if err != nil {
		return err
	}
	s.Data[AWSSecretAccessKey] = secret.Value()

	return nil
}

func (c AWS) AccessKeyID() string {
	return c.Data[AWSAccessKeyID]
}

func (c AWS) SecretAccessKey() string {
	return c.Data[AWSSecretAccessKey]
}
