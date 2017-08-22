package cmds

import (
	"flag"
	"log"

	v "github.com/appscode/go/version"
	"github.com/appscode/pharmer/config"
	cfgCmd "github.com/appscode/pharmer/config/cmds"
	credCmd "github.com/appscode/pharmer/credential/cmds"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewRootCmd(version string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:               "pharmer [command]",
		Short:             `Pharmer by Appscode - Manages farms`,
		DisableAutoGenTag: true,
		PersistentPreRunE: func(c *cobra.Command, args []string) error {
			c.Flags().VisitAll(func(flag *pflag.Flag) {
				log.Printf("FLAG: --%s=%q", flag.Name, flag.Value)
			})

			return config.CreateDefaultConfigIfAbsent()
		},
	}
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	// ref: https://github.com/kubernetes/kubernetes/issues/17162#issuecomment-225596212
	flag.CommandLine.Parse([]string{})

	rootCmd.AddCommand(credCmd.NewCmdCredential())
	rootCmd.AddCommand(cfgCmd.NewCmdConfig())
	rootCmd.AddCommand(NewCmdCreate())
	rootCmd.AddCommand(NewCmdDelete())
	rootCmd.AddCommand(NewCmdReconfigure())
	rootCmd.AddCommand(NewCmdList())
	rootCmd.AddCommand(NewCmdUse())
	rootCmd.AddCommand(NewCmdSSH())
	rootCmd.AddCommand(NewCmdBackup())
	rootCmd.AddCommand(v.NewCmdVersion())

	return rootCmd
}
