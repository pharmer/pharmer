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
	"pharmer.dev/pharmer/cloud"
	"pharmer.dev/pharmer/store"

	"github.com/appscode/go/signals"
	"github.com/appscode/go/term"
	"github.com/spf13/cobra"
	"k8s.io/klog/klogr"
	clusterapis "sigs.k8s.io/cluster-api/pkg/apis"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// TODO: make sure it works
func newCmdController() *cobra.Command {
	var provider, clusterName string

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

			storeProvider, err := store.GetStoreProvider(cmd)
			term.ExitOnError(err)

			cluster, err := storeProvider.Clusters().Get(clusterName)
			if err != nil {
				term.Fatalln(err)
			}

			scope := cloud.NewScope(cloud.NewScopeParams{
				Cluster:       cluster,
				StoreProvider: storeProvider,
				Logger:        klogr.New().WithValues("cluster-name", clusterName),
			})

			cm, err := scope.GetCloudManager()
			if err != nil {
				term.Fatalln(err)
			}

			err = cm.SetCloudConnector() //Connector()
			if err != nil {
				term.Fatalln(err)
			}

			//err = cm.InitializeMachineActuator(mgr)
			//if err != nil {
			//	term.Fatalln(err)
			//}

			if err := clusterapis.AddToScheme(mgr.GetScheme()); err != nil {
				term.Fatalln(err)
			}

			if err := cm.AddToManager(mgr); err != nil {
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
	cmd.Flags().StringVar(&clusterName, "cluster-name", clusterName, "Cluster name")

	return cmd
}
