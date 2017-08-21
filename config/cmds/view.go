package cmds

import (
	"fmt"
	"os"

	otx "github.com/appscode/pharmer/config"
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
)

func newCmdView() *cobra.Command {
	var configPath string
	setCmd := &cobra.Command{
		Use:               "view",
		Short:             "Print Pharmer config",
		Example:           "Pharmer config view",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				cmd.Help()
				os.Exit(1)
			}
			if configPath == "" {
				configPath = otx.DefaultConfigPath()
			}
			return viewContext(configPath)
		},
	}

	setCmd.Flags().StringVar(&configPath, "provider", "", "Path to Pharmer config file")
	return setCmd
}

func viewContext(configPath string) error {
	config, err := otx.LoadConfig(configPath)
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	fmt.Println(string(data))
	return nil
}
