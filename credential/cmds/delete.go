package cmds

import (
	go_ctx "context"
	"fmt"
	"os"

	"github.com/appscode/log"
	"github.com/appscode/pharmer/config"
	"github.com/appscode/pharmer/context"
	"github.com/spf13/cobra"
)

func NewCmdDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "delete",
		Short:             "Delete a cloud credential",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				fmt.Fprintf(os.Stderr, "You can only specify one argument, found %d", len(args))
				cmd.Help()
				os.Exit(1)
			}

			cfgFile, _ := config.GetConfigFile(cmd.Flags())
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				log.Fatalln(err)
			}
			store := context.NewStoreProvider(go_ctx.TODO(), cfg)
			err = store.Credentials().Delete(args[0])
			if err != nil {
				log.Fatalln(err)
			}
		},
	}
	return cmd
}
