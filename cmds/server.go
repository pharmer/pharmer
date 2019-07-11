package cmds

import (
	"fmt"

	"github.com/appscode/go/term"
	"github.com/nats-io/stan.go"
	"github.com/pharmer/pharmer/apiserver"
	"github.com/pharmer/pharmer/config"
	"github.com/pharmer/pharmer/store"
	"github.com/spf13/cobra"
)

func newCmdServer() *cobra.Command {
	//opts := options.NewClusterCreateConfig()
	var natsurl string
	var clientid string
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

			err = runServer(storeProvider, conn)

			//err = http.ListenAndServe(":4155", route(ctx, conn))
			term.ExitOnError(err)
			<-make(chan interface{})
		},
	}
	cmd.Flags().StringVar(&natsurl, "nats-url", "nats://localhost:4222", "Nats streaming server url")
	cmd.Flags().StringVar(&clientid, "nats-client-id", "worker-p", "Nats streaming server client id")

	return cmd
}

//const ClientID = "worker-x"

func runServer(storeProvider store.Interface, conn stan.Conn) error {

	//defer apiserver.LogCloser(conn)

	server := apiserver.New(conn)
	err := server.CreateCluster(storeProvider)
	if err != nil {
		return err
	}

	if err = server.DeleteCluster(storeProvider); err != nil {
		return err
	}

	return server.RetryCluster(storeProvider)
}
