package cmds

import (
	"github.com/appscode/go/term"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/store"
	"github.com/spf13/cobra"
	"k8s.io/sample-controller/pkg/signals"
	clusterapis "sigs.k8s.io/cluster-api/pkg/apis"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// TODO: make sure it works
func newCmdController() *cobra.Command {
	var ownerID, provider, clusterName string

	machineSetupConfig := "/etc/machinesetup/machine_setup_configs.yaml"
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

			err = store.SetProvider(cmd, ownerID)
			if err != nil {
				term.Fatalln(err)
			}

			cluster, err := store.StoreProvider.Clusters().Get(clusterName)
			if err != nil {
				term.Fatalln(err)
			}

			cm, err := cloud.GetCloudManager(cluster)
			if err != nil {
				term.Fatalln(err)
			}

			err = cm.InitializeMachineActuator(mgr)
			if err != nil {
				term.Fatalln(err)
			}

			if err := clusterapis.AddToScheme(mgr.GetScheme()); err != nil {
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
	cmd.Flags().StringVar(&clusterName, "cluster-na", clusterName, "Cluster name")
	cmd.Flags().StringVarP(&ownerID, "owner", "o", ownerID, "Current user id")

	return cmd
}
