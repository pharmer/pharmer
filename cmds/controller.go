package cmds

import (
	"context"
	"fmt"

	"github.com/appscode/go/term"
	"github.com/pharmer/pharmer/cloud"
	pharmerConf "github.com/pharmer/pharmer/config"
	"github.com/spf13/cobra"
	clusterapis "sigs.k8s.io/cluster-api/pkg/apis"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

func newCmdController() *cobra.Command {
	//s := config.
	ownerID := ""
	machineSetupConfig := "/etc/machinesetup/machine_setup_configs.yaml"
	provider := "digitalocean"
	cmd := &cobra.Command{
		Use:               "controller",
		Short:             "Bootstrap as a Kubernetes master or node",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			conf := config.GetConfigOrDie()
			mgr, err := manager.New(conf, manager.Options{})
			if err != nil {
				term.Fatalln(err)
			}

			// Initialize cluster actuator.

			cfgFile, _ := pharmerConf.GetConfigFile(cmd.Flags())
			cfg, err := pharmerConf.LoadConfig(cfgFile)
			term.ExitOnError(err)

			fmt.Println(provider)

			ctx := cloud.NewContext(context.Background(), cfg, pharmerConf.GetEnv(cmd.Flags()))

			cm, err := cloud.GetCloudManager(provider, ctx)
			term.ExitOnError(err)

			err = cm.InitializeMachineActuator(mgr)
			term.ExitOnError(err)

			if err := clusterapis.AddToScheme(mgr.GetScheme()); err != nil {
				term.Fatalln(err)
			}
			cm.SetOwner(ownerID)

			if err := cm.AddToManager(ctx, mgr); err != nil {
				term.Fatalln(err)
			}

			if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
				term.Printf("Failed to run manager: %v", err)
			}
		},
	}
	//s.AddFlags(cmd.Flags())
	cmd.Flags().StringVar(&machineSetupConfig, "machine-setup-config", machineSetupConfig, "path to the machine setup config")
	cmd.Flags().StringVar(&provider, "provider", provider, "Cloud provider name")
	cmd.Flags().StringVarP(&ownerID, "owner", "o", ownerID, "Current user id")

	return cmd
}
