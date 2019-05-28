package cmds

import (
	"github.com/spf13/cobra"
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
			//conf := config.GetConfigOrDie()
			//mgr, err := manager.New(conf, manager.Options{})
			//if err != nil {
			//	term.Fatalln(err)
			//}
			//
			//// Initialize cluster actuator.
			//
			//cfgFile, _ := pharmerConf.GetConfigFile(cmd.Flags())
			//cfg, err := pharmerConf.LoadConfig(cfgFile)
			//term.ExitOnError(err)
			//
			//ctx := cloud.NewContext(context.Background(), cfg, pharmerConf.GetEnv(cmd.Flags()))
			//
			////todo:fix
			//cm, err := cloud.GetCloudManager(provider, nil, nil)
			//term.ExitOnError(err)
			//
			//err = cm.InitializeMachineActuator(mgr)
			//term.ExitOnError(err)
			//
			//if err := clusterapis.AddToScheme(mgr.GetScheme()); err != nil {
			//	term.Fatalln(err)
			//}
			//cm.SetOwner(ownerID)
			//
			//if err := cm.AddToManager(ctx, mgr); err != nil {
			//	term.Fatalln(err)
			//}
			//
			//if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
			//	term.Printf("Failed to run manager: %v", err)
			//}
		},
	}
	//s.AddFlags(cmd.Flags())
	cmd.Flags().StringVar(&machineSetupConfig, "machine-setup-config", machineSetupConfig, "path to the machine setup config")
	cmd.Flags().StringVar(&provider, "provider", provider, "Cloud provider name")
	cmd.Flags().StringVarP(&ownerID, "owner", "o", ownerID, "Current user id")

	return cmd
}
