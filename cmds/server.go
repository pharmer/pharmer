package cmds

import (
	"context"
	"fmt"

	"github.com/appscode/go/term"
	stan "github.com/nats-io/stan.go"
	"github.com/pharmer/pharmer/apiserver"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/config"
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
			/*if err := opts.ValidateFlags(cmd, args); err != nil {
				term.Fatalln(err)
			}*/

			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				term.Fatalln(err)
			}
			if cfg.Store.Postgres == nil {
				term.Fatalln("Use postgres as storage provider")
			}
			fmt.Println(natsurl)

			ctx := cloud.NewContext(context.Background(), cfg, config.GetEnv(cmd.Flags()))

			conn, err := stan.Connect(
				"pharmer-cluster",
				clientid,
				stan.NatsURL(natsurl),
			)
			defer apiserver.LogCloser(conn)
			term.ExitOnError(err)

			err = runServer(ctx, conn)

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

func runServer(ctx context.Context, conn stan.Conn) error {

	//defer apiserver.LogCloser(conn)

	server := apiserver.New(ctx, conn)
	err := server.CreateCluster()
	if err != nil {
		return err
	}

	if err = server.DeleteCluster(); err != nil {
		return err
	}

	return server.RetryCluster()
}
