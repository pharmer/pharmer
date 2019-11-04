/*
Copyright The Pharmer Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmds

import (
	"flag"
	"io"
	"os"
	"strings"

	_ "pharmer.dev/pharmer/cloud/providers"
	cpCmd "pharmer.dev/pharmer/cmds/cloud"
	"pharmer.dev/pharmer/config"
	cfgCmd "pharmer.dev/pharmer/config/cmds"
	_ "pharmer.dev/pharmer/store/providers"

	"github.com/appscode/go/analytics"
	"github.com/appscode/go/term"
	v "github.com/appscode/go/version"
	ga "github.com/jpillora/go-ogle-analytics"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/klog"
	"kmodules.xyz/client-go/logs"
)

const (
	gaTrackingCode = "UA-62096468-20"
)

func NewRootCmd(in io.Reader, out, errwriter io.Writer, version string) *cobra.Command {
	var (
		enableAnalytics = true
	)
	rootCmd := &cobra.Command{
		Use:               "pharmer [command]",
		Short:             `Pharmer by Appscode - Kubernetes Cluster Manager for Kubeadm`,
		DisableAutoGenTag: true,
		PersistentPreRunE: func(c *cobra.Command, args []string) error {
			c.Flags().VisitAll(func(flag *pflag.Flag) {
				klog.V(1).Infof("FLAG: --%s=%q", flag.Name, flag.Value)
			})
			if enableAnalytics && gaTrackingCode != "" {
				if client, err := ga.NewClient(gaTrackingCode); err == nil {
					client.ClientID(analytics.ClientID())
					parts := strings.Split(c.CommandPath(), " ")
					err = client.Send(ga.NewEvent(parts[0], strings.Join(parts[1:], "/")).Label(version))
					if err != nil {
						return err
					}
				}
			}

			if cfgFile, setByUser := config.GetConfigFile(c.Flags()); !setByUser {
				if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
					return config.Save(config.NewDefaultConfig(), cfgFile)
				}
			}
			logs.ParseFlags()

			return nil
		},
	}
	config.AddFlags(rootCmd.PersistentFlags())
	rootCmd.PersistentFlags().BoolVar(&enableAnalytics, "analytics", enableAnalytics, "Send analytical events to Google Guard")
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	// ref: https://github.com/kubernetes/kubernetes/issues/17162#issuecomment-225596212
	err := flag.CommandLine.Parse([]string{})
	if err != nil {
		term.Fatalln(err)
	}

	rootCmd.AddCommand(newCmdCreate())
	rootCmd.AddCommand(newCmdGet(out))
	rootCmd.AddCommand(newCmdDelete())
	rootCmd.AddCommand(newCmdDescribe(out))
	rootCmd.AddCommand(newCmdEdit(out, errwriter))
	rootCmd.AddCommand(newCmdBackup())
	rootCmd.AddCommand(newCmdUse())
	rootCmd.AddCommand(newCmdSSH())

	rootCmd.AddCommand(cfgCmd.NewCmdConfig())

	rootCmd.AddCommand(v.NewCmdVersion())

	rootCmd.AddCommand(cpCmd.NewCmdApply())
	rootCmd.AddCommand(newCmdController())

	rootCmd.AddCommand(newCmdServer())

	return rootCmd
}
