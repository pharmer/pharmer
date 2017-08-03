package cmd

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/appscode/appctl/pkg/util"
	term "github.com/appscode/go-term"
	"github.com/appscode/go/flags"
	"github.com/appscode/pharmer/commissioner"
	"github.com/spf13/cobra"
)

func NewCmdClusterCreate() *cobra.Command {
	var provider, name, version, cloudCred string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Cluster create commisioning",
		Run: func(cmd *cobra.Command, args []string) {
			flags.SetLogLevel(4)
			flags.EnsureRequiredFlags(cmd, "provider")
			if len(args) > 0 {
				name = args[0]
			} else {
				term.Fatalln("missing cluster name")
			}
			c, err := commissioner.NewComissionar(provider, name)
			if cloudCred == "" {
				reader := bufio.NewReader(os.Stdin)
				data, err := ioutil.ReadAll(reader)
				term.ExitOnError(err)
				c.Credential, err = util.ParseCloudCredential(string(data), provider)
				term.ExitOnError(err)
			} else {
				c.CredentialPHID = cloudCred
			}
			err = c.ClusterCreate(version)
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
		},
	}
	cmd.Flags().StringVarP(&provider, "provider", "p", "", "Give the provider Name")
	cmd.Flags().StringVar(&version, "version", "", "Give the cluster version")
	cmd.Flags().StringVar(&cloudCred, "cloud-credential", "", "Use preconfigured cloud credential phid")
	return cmd
}
