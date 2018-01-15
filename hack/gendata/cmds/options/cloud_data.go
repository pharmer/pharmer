package options

import (
	"github.com/spf13/pflag"
	"github.com/spf13/cobra"
	"github.com/appscode/go/flags"
)

const (
	DefaultKubernetesVersion string = "1.8.0"
)

type CloudData struct {
	Provider       string
	//credential file for gce
	CredentialFile string
	//access token for digitalocean
	DoToken        string

	GCEProjectName string
	KubernetesVersions string
}



func NewCloudData() *CloudData {
	return &CloudData{
		Provider: "",
		KubernetesVersions: DefaultKubernetesVersion,
	}
}

func (c *CloudData) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&c.Provider, "provider", "p", c.Provider, "Name of the Cloud provider")
	fs.StringVarP(&c.CredentialFile, "credential-file", "c", c.CredentialFile, "Location of cloud credential file (required when --provider=gce)")
	fs.StringVar(&c.GCEProjectName, "google-project", c.GCEProjectName, "When using the Google provider, specify the Google project (required when --provider=gce)")
	fs.StringVar(&c.KubernetesVersions, "versions-support", c.KubernetesVersions, "Supported versions of kubernetes, example: --versions-support=1.1.0,1.2.0")
	fs.StringVar(&c.DoToken, "do-token", c.DoToken, "provide this flag when provider is digitalocean")
}

func (c *CloudData) ValidateFlags(cmd *cobra.Command, args []string) error {
	var ensureFlags []string
	switch c.Provider {
	case "gce":
		ensureFlags = []string{"provider",  "credential-file", "google-project"}
		break
	case "digitalocean":
		ensureFlags = []string{"provider",  "do-token"}
		break
	default:
		ensureFlags = []string{"provider"}
		break
	}

	flags.EnsureRequiredFlags(cmd, ensureFlags...)

	return nil
}
