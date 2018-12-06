package cmds

import (
	"flag"
	"log"
	"strings"

	"github.com/appscode/go/analytics"
	v "github.com/appscode/go/version"
	ga "github.com/jpillora/go-ogle-analytics"
	_ "github.com/pharmer/pharmer/cloud/providers"
	"github.com/pharmer/pharmer/config"
	_ "github.com/pharmer/pharmer/store/providers"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	gaTrackingCode = "UA-62096468-20"
)

func NewRootCmd(version string) *cobra.Command {
	var (
		enableAnalytics   = true
		analyticsClientID string
	)
	rootCmd := &cobra.Command{
		Use:               "pharmer-tools",
		Short:             `Pharmer by Appscode - Manages farms`,
		DisableAutoGenTag: true,
		PersistentPreRun: func(c *cobra.Command, args []string) {
			c.Flags().VisitAll(func(flag *pflag.Flag) {
				log.Printf("FLAG: --%s=%q", flag.Name, flag.Value)
			})
			if enableAnalytics && gaTrackingCode != "" {
				if client, err := ga.NewClient(gaTrackingCode); err == nil {
					analyticsClientID = analytics.ClientID()
					client.ClientID(analyticsClientID)
					parts := strings.Split(c.CommandPath(), " ")
					client.Send(ga.NewEvent(parts[0], strings.Join(parts[1:], "/")).Label(version))
				}
			}
		},
	}
	config.AddFlags(rootCmd.PersistentFlags())
	rootCmd.PersistentFlags().BoolVar(&enableAnalytics, "analytics", enableAnalytics, "Send analytical events to Google Guard")
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	// ref: https://github.com/kubernetes/kubernetes/issues/17162#issuecomment-225596212
	flag.CommandLine.Parse([]string{})

	rootCmd.AddCommand(NewCmdGenData())
	rootCmd.AddCommand(NewCmdKubeSupport())
	rootCmd.AddCommand(NewCmdGenNPM())
	rootCmd.AddCommand(v.NewCmdVersion())

	return rootCmd
}
