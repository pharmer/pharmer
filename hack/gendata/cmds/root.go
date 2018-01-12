package cmds

import (
	"flag"
	"fmt"

	"github.com/appscode/go/term"
	"github.com/pharmer/pharmer/hack/gendata/cmds/options"
	"github.com/pharmer/pharmer/hack/gendata/providers"
	"github.com/spf13/cobra"
)

func NewCmdLoadData() *cobra.Command {
	opts := options.NewCloudData()
	cmd := &cobra.Command{
		Use:               "gendata",
		Short:             "Load Kubernetes cluster data for a given cloud provider",
		Example:           "",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.ValidateFlags(cmd, args); err != nil {
				term.Fatalln(err)
			}
			// code here
			switch opts.Provider {
			case "gce":
				cloudProvider := providers.NewCloudProvider()
				gceClient, err := cloudProvider.Gce(opts.GCEProjectName, opts.Config, opts.KubernetesVersions)
				if err != nil {
					term.Fatalln(err)
				}
				err = providers.WriteCloudData(gceClient)
				if err != nil {
					term.Fatalln(err)
				} else {
					term.Successln("Data successfully written in data/gce/cloud.json for gce")
				}
				break
			default:
				term.Fatalln(fmt.Errorf("Valid/Supported provider name required"))
			}
		},
	}
	cmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	opts.AddFlags(cmd.Flags())
	return cmd
}
