package credential

import (
	"github.com/pharmer/cloud/pkg/apis"
	v1 "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	"github.com/spf13/pflag"
	"gopkg.in/ini.v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AWS struct {
	CommonSpec

	region          string
	accessKeyID     string
	secretAccessKey string
}

func NewAWS() *AWS {
	return &AWS{}
}

func (c AWS) Region() string          { return get(c.Data, AWSRegion, c.region) }
func (c AWS) AccessKeyID() string     { return get(c.Data, AWSAccessKeyID, c.accessKeyID) }
func (c AWS) SecretAccessKey() string { return get(c.Data, AWSSecretAccessKey, c.secretAccessKey) }

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

func (c *AWS) LoadFromEnv() {
	c.CommonSpec.LoadFromEnv(c.Format())
}

func (c AWS) IsValid() (bool, error) {
	return c.CommonSpec.IsValid(c.Format())
}

func (c *AWS) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.region, "aws.region", c.region, "provide this flag when provider is aws")
	fs.StringVar(&c.accessKeyID, apis.AWS+"."+AWSAccessKeyID, c.accessKeyID, "provide this flag when provider is aws")
	fs.StringVar(&c.secretAccessKey, apis.AWS+"."+AWSSecretAccessKey, c.secretAccessKey, "provide this flag when provider is aws")
}

func (c AWS) RequiredFlags() []string {
	return []string{
		apis.AWS + "." + "region",
		apis.AWS + "." + AWSAccessKeyID,
		apis.AWS + "." + AWSSecretAccessKey,
	}
}

func (c AWS) Format() v1.CredentialFormat {
	return v1.CredentialFormat{
		ObjectMeta: metav1.ObjectMeta{
			Name: apis.AWS,
			Labels: map[string]string{
				apis.KeyCloudProvider: apis.AWS,
			},
			Annotations: map[string]string{
				apis.KeyClusterCredential: "",
				apis.KeyDNSCredential:     "",
				apis.KeyStorageCredential: "",
			},
		},
		Spec: v1.CredentialFormatSpec{
			Provider:      apis.AWS,
			DisplayFormat: "field",
			Fields: []v1.CredentialField{
				{
					Envconfig: "AWS_ACCESS_KEY_ID",
					Form:      "aws_access_key_id",
					JSON:      AWSAccessKeyID,
					Label:     "Access Key Id",
					Input:     "text",
				},
				{
					Envconfig: "AWS_SECRET_ACCESS_KEY",
					Form:      "aws_secret_access_key",
					JSON:      AWSSecretAccessKey,
					Label:     "Secret Access Key",
					Input:     "password",
				},
			},
		},
	}
}
