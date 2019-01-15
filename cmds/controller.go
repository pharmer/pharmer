package cmds

import (
	"github.com/spf13/cobra"
	"sigs.k8s.io/cluster-api/pkg/controller/config"
)

func newCmdController() *cobra.Command {
	s := config.ControllerConfig
	provider := "digitalocean"
	cmd := &cobra.Command{
		Use:               "controller",
		Short:             "Bootstrap as a Kubernetes master or node",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			/*conf, err := controller.GetConfig(s.Kubeconfig)
			if err != nil {
				term.Fatalln(err)
			}
			client, err := clientset.NewForConfig(conf)
			if err != nil {
				term.Fatalln(err)
			}

			cfgFile, _ := pharmerConf.GetConfigFile(cmd.Flags())
			cfg, err := pharmerConf.LoadConfig(cfgFile)
			term.ExitOnError(err)

			ctx := cloud.NewContext(context.Background(), cfg, pharmerConf.GetEnv(cmd.Flags()))
			cm, err := cloud.GetCloudManager(provider, ctx)
			term.ExitOnError(err)

			err = cm.InitializeActuator(client.ClusterV1alpha1().Machines(core.NamespaceDefault))
			term.ExitOnError(err)*/

			//actuator, err :=
			//	shutdown := make(chan struct{})
			//si := sharedinformers.NewSharedInformers(conf, shutdown)
			//	mc := machineset.NewMachineSetController(conf, si)
			//go mc.Run(make(chan struct{}))

			//	c := machine.NewMachineController(conf, si, cm)
			//c.RunAsync(shutdown)

			select {}
		},
	}
	s.AddFlags(cmd.Flags())
	cmd.Flags().StringVar(&provider, "provider", provider, "Cloud provider name")
	return cmd
}
