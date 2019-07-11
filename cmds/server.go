package cmds

import (
	"flag"
	"fmt"

	"github.com/appscode/go/term"
	"github.com/nats-io/stan.go"
	"github.com/pharmer/pharmer/apiserver"
	"github.com/pharmer/pharmer/config"
	"github.com/pharmer/pharmer/store"
	"github.com/spf13/cobra"
	natslogr "gomodules.xyz/nats-logr"
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
			fmt.Println(natsurl)

			conn, err := stan.Connect(
				"pharmer-cluster",
				clientid,
				stan.NatsURL(natsurl),
			)
			defer apiserver.LogCloser(conn)
			term.ExitOnError(err)

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
