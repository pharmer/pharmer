package credential

import (
	"github.com/pharmer/cloud/pkg/apis"
	v1 "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AzureStorage struct {
	CommonSpec

	account string
	key     string
}

func (c AzureStorage) Account() string { return c.Data[AzureStorageAccount] }
func (c AzureStorage) Key() string     { return c.Data[AzureStorageKey] }

func (c *AzureStorage) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.account, apis.AzureStorage+"."+AzureStorageAccount, c.account, "Azure storage account")
	fs.StringVar(&c.key, apis.AzureStorage+"."+AzureStorageKey, c.key, "Azure storage account key")
}

func (_ AzureStorage) RequiredFlags() []string {
	return []string{
		apis.AzureStorage + "." + AzureStorageAccount,
		apis.AzureStorage + "." + AzureStorageKey,
	}
}

func (_ AzureStorage) Format() v1.CredentialFormat {
	return v1.CredentialFormat{
		ObjectMeta: metav1.ObjectMeta{
			Name: apis.Azure + "-storage-cred",
			Labels: map[string]string{
				apis.KeyCloudProvider: apis.Azure,
			},
			Annotations: map[string]string{
				apis.KeyStorageCredential: "",
			},
		},
		Spec: v1.CredentialFormatSpec{
			Provider:      apis.Azure,
			DisplayFormat: "field",
			Fields: []v1.CredentialField{
				{
					Envconfig: "AZURE_STORAGE_ACCOUNT",
					Form:      "azure_storage_account",
					JSON:      AzureStorageAccount,
					Label:     "Azure Storage Account",
					Input:     "text",
				},
				{
					Envconfig: "AZURE_STORAGE_KEY",
					Form:      "azure_storage_key",
					JSON:      AzureStorageKey,
					Label:     "Azure Storage Account Key",
					Input:     "password",
				},
			},
		},
	}
}
