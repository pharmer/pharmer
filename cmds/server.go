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

	"pharmer.dev/pharmer/apiserver"
	"pharmer.dev/pharmer/config"
	"pharmer.dev/pharmer/store"

	"github.com/appscode/go/term"
	"github.com/nats-io/stan.go"
	"github.com/spf13/cobra"
	natslogr "gomodules.xyz/nats-logr"
	"k8s.io/klog/klogr"
)

var (
	log = klogr.New().WithName("[pharmer-serve]")
)

func newCmdServer() *cobra.Command {
	//opts := options.NewClusterCreateConfig()
	var natsurl string
	var clientid string
	var logToNats bool
	cmd := &cobra.Command{
		Use:               "serve",
		Short:             "Pharmer apiserver",
		Example:           "pharmer serve",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				term.Fatalln(err)
			}
			if cfg.Store.Postgres == nil {
				term.Fatalln("Use postgres as storage provider")
			}

			conn, err := stan.Connect(
				"pharmer-cluster",
				clientid,
				stan.NatsURL(natsurl),
			)
			defer apiserver.LogCloser(conn)
			term.ExitOnError(err)

			log.Info("Connected to nats streaming server", "natsurl", natsurl, "clusterID", "pharmer-cluster", "clientID", clientid)

			storeProvider, err := store.NewStoreInterface(cfg)
			term.ExitOnError(err)

			err = runServer(storeProvider, conn, natsurl, logToNats)

			//err = http.ListenAndServe(":4155", route(ctx, conn))
			term.ExitOnError(err)
			<-make(chan interface{})
		},
	}
	_ = flag.CommandLine.Parse([]string{})

	natsLogFlags := flag.NewFlagSet("nats-log-flags", flag.ExitOnError)
	natslogr.InitFlags(natsLogFlags)

	flag.VisitAll(func(f1 *flag.Flag) {
		f2 := natsLogFlags.Lookup(f1.Name)
		if f2 != nil {
			_ = f2.Value.Set(f1.Value.String())
		}
	})

	cmd.Flags().StringVar(&natsurl, "nats-url", "nats://localhost:4222", "Nats streaming server url")
	cmd.Flags().StringVar(&clientid, "nats-client-id", "worker-p", "Nats streaming server client id")
	cmd.Flags().BoolVar(&logToNats, "logToNats", false, "Publish logs to nats streaming server")

	return cmd
}

//const ClientID = "worker-x"

func runServer(storeProvider store.Interface, conn stan.Conn, natsurl string, logToNats bool) error {

	//defer apiserver.LogCloser(conn)

	server := apiserver.New(conn)
	err := server.CreateCluster(storeProvider, natsurl, logToNats)
	if err != nil {
		return err
	}

	if err = server.DeleteCluster(storeProvider, natsurl, logToNats); err != nil {
		return err
	}

	return server.RetryCluster(storeProvider, natsurl, logToNats)
}
