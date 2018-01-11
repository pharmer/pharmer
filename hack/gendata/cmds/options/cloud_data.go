package options

import (
	"github.com/spf13/pflag"
	"github.com/spf13/cobra"
	"github.com/appscode/go/flags"
)

type CloudData struct {
	Provider string
	Config string //file

	GCEProjectName string
}



func NewCloudData() *CloudData {
	return &CloudData{
		Provider: "",
	}
}

func (c *CloudData) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&c.Provider, "provider", "p", c.Provider, "Name of the Cloud provider")
	fs.StringVarP(&c.Config, "config", "c", c.Config, "Location fo cloud credential file")
	fs.StringVar(&c.GCEProjectName, "google-project", c.GCEProjectName, "When using the Google provider, specify the Google project (required when --provider=google")
}

func (c *CloudData) ValidateFlags(cmd *cobra.Command, args []string) error {
	var ensureFlags []string
	switch c.Provider {
	case "gce":
		ensureFlags = []string{"provider",  "config", "google-project"}
		break
	default:
		break
	}

	flags.EnsureRequiredFlags(cmd, ensureFlags...)

	return nil
}
