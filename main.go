package main

import (
	//"github.com/appscode/pharmer/errorhandlers"
	"flag"
	"log"
	_ "net/http/pprof"
	"os"

	"github.com/appscode/client/cli"
	term "github.com/appscode/go-term"
	_env "github.com/appscode/go/env"
	clustercmd "github.com/appscode/pharmer/commissioner/cmd"
	//logs "github.com/appscode/log/golog"
	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use: "commissioner",
		PersistentPreRun: func(c *cobra.Command, args []string) {
			a, loggedIn := cli.GetAuthOrAnon()
			if a.Env != _env.Prod {
				term.Warningln("Using env:", a.Env)
			}
			if loggedIn {
				term.Infoln("Logged into team:", a.TeamAddr())
			}
		},
	}
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	rootCmd.AddCommand(clustercmd.NewCmdCluster())

	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
	os.Exit(0)
}
