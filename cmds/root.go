package cmds

import (
	"flag"
	"io"
	"log"
	"os"
	"strings"

	v "github.com/appscode/go/version"
	cpCmd "github.com/appscode/pharmer/cloud/cmds"
	_ "github.com/appscode/pharmer/cloud/providers"
	"github.com/appscode/pharmer/config"
	cfgCmd "github.com/appscode/pharmer/config/cmds"
	"github.com/appscode/pharmer/data/files"
	_ "github.com/appscode/pharmer/store/providers"
	"github.com/jpillora/go-ogle-analytics"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	gaTrackingCode = "UA-62096468-20"
)

func NewRootCmd(in io.Reader, out, err io.Writer, version string) *cobra.Command {
	var (
		enableAnalytics = true
	)
	rootCmd := &cobra.Command{
		Use:               "pharmer [command]",
		Short:             `Pharmer by Appscode - Manages farms`,
		DisableAutoGenTag: true,
		PersistentPreRunE: func(c *cobra.Command, args []string) error {
			c.Flags().VisitAll(func(flag *pflag.Flag) {
				log.Printf("FLAG: --%s=%q", flag.Name, flag.Value)
			})
			if enableAnalytics && gaTrackingCode != "" {
				if client, err := ga.NewClient(gaTrackingCode); err == nil {
					parts := strings.Split(c.CommandPath(), " ")
					client.Send(ga.NewEvent(parts[0], strings.Join(parts[1:], "/")).Label(version))
				}
			}

			files.Load(config.GetEnv(c.Flags()))

			if cfgFile, setByUser := config.GetConfigFile(c.Flags()); !setByUser {
				if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
					return config.Save(config.NewDefaultConfig(), cfgFile)
				}
			}
			return nil
		},
	}
	config.AddFlags(rootCmd.PersistentFlags())
	rootCmd.PersistentFlags().BoolVar(&enableAnalytics, "analytics", enableAnalytics, "Send analytical events to Google Guard")
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	// ref: https://github.com/kubernetes/kubernetes/issues/17162#issuecomment-225596212
	flag.CommandLine.Parse([]string{})

	rootCmd.AddCommand(newCmdCreate())
	rootCmd.AddCommand(newCmdGet(out))
	rootCmd.AddCommand(newCmdDelete())
	rootCmd.AddCommand(newCmdIssue())
	rootCmd.AddCommand(newCmdDescribe(out))
	rootCmd.AddCommand(newCmdEdit(out, err))
	rootCmd.AddCommand(newCmdBackup())
	rootCmd.AddCommand(newCmdUse())
	rootCmd.AddCommand(newCmdSSH())
	rootCmd.AddCommand(NewCmdInspector())

	rootCmd.AddCommand(cfgCmd.NewCmdConfig())
	rootCmd.AddCommand(cpCmd.NewCmdApply())
	rootCmd.AddCommand(v.NewCmdVersion())

	return rootCmd
}
