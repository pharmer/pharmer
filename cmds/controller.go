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

			/*client, err := clientset.NewForConfig(conf)
			if err != nil {
				term.Fatalln(err)
			}*/

			// Initialize event recorder.
			//record.InitFromRecorder(mgr.GetRecorder("pharmer-controller"))

			// Initialize cluster actuator.

			cfgFile, _ := pharmerConf.GetConfigFile(cmd.Flags())
			cfg, err := pharmerConf.LoadConfig(cfgFile)
			term.ExitOnError(err)

			fmt.Println(provider)


			ctx := cloud.NewContext(context.Background(), cfg, pharmerConf.GetEnv(cmd.Flags()))

			/*configWatch, err := machinesetup.NewConfigWatch(machineSetupConfig)
			term.ExitOnError(err)
			ctx = cloud.WithMachineSetup(ctx, configWatch)*/

			cm, err := cloud.GetCloudManager(provider, ctx)
			term.ExitOnError(err)

			err = cm.InitializeMachineActuator(mgr)
			term.ExitOnError(err)

			/*err = cm.InitializeActuator(client.ClusterV1alpha1(), mgr.GetRecorder("pharmer-controller"), mgr.GetScheme())
			term.ExitOnError(err)*/

			/*if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
				klog.Fatal(err)
			}
			*/
			if err := clusterapis.AddToScheme(mgr.GetScheme()); err != nil {
				term.Fatalln(err)
			}

			if err := cm.AddToManager(ctx, mgr); err != nil {
				term.Fatalln(err)
			}

			//actuator, err :=
			//	shutdown := make(chan struct{})
			//si := sharedinformers.NewSharedInformers(conf, shutdown)
			//	mc := machineset.NewMachineSetController(conf, si)
			//go mc.Run(make(chan struct{}))

			//	c := machine.NewMachineController(conf, si, cm)
			//c.RunAsync(shutdown)

			if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
				term.Printf("Failed to run manager: %v", err)
			}
		},
	}
	//s.AddFlags(cmd.Flags())
	cmd.Flags().StringVar(&machineSetupConfig, "machine-setup-config", machineSetupConfig, "path to the machine setup config")
	cmd.Flags().StringVar(&provider, "provider", provider, "Cloud provider name")
	return cmd
}
