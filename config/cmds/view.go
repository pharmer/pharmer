package cmds

import (
	"os"

	"github.com/appscode/go-term"
	otx "github.com/appscode/pharmer/config"
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
)

func newCmdView() *cobra.Command {
	setCmd := &cobra.Command{
		Use:               "view",
		Short:             "Print pharmer config",
		Example:           "pharmer config view",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				cmd.Help()
				os.Exit(1)
			}
			viewContext(otx.GetConfigPath(cmd))
		},
	}
	return setCmd
}

func viewContext(configPath string) {
	config, err := otx.LoadConfig(configPath)
	term.ExitOnError(err)

	data, err := yaml.Marshal(config)
	term.ExitOnError(err)

	term.Infoln(string(data))
}
