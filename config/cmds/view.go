package cmds

import (
	"fmt"
	"os"
	"github.com/ghodss/yaml"
	"github.com/pharmer/pharmer/config"
	"github.com/spf13/cobra"
)

func newCmdView() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "view",
		Short:             "Print Pharmer config",
		Example:           "Pharmer config view",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				cmd.Help()
				os.Exit(1)
			}

			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				return err
			}
			data, err := yaml.Marshal(cfg)
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		},
	}
	return cmd
}
