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
package cloud

import (
	"pharmer.dev/pharmer/cloud"
	"pharmer.dev/pharmer/cmds/cloud/options"
	"pharmer.dev/pharmer/store"

	"github.com/appscode/go/term"
	"github.com/spf13/cobra"
)

func NewCmdUse() *cobra.Command {
	opts := options.NewClusterUseConfig()
	cmd := &cobra.Command{
		Use:               "cluster",
		Short:             "Sets `kubectl` context to given cluster",
		Example:           `pharmer use cluster <name>`,
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.ValidateFlags(cmd, args); err != nil {
				term.Fatalln(err)
			}

			storeProvider, err := store.GetStoreProvider(cmd)
			if err != nil {
				term.Fatalln(err)
			}

			cluster, err := storeProvider.Clusters().Get(opts.ClusterName)
			if err != nil {
				term.Fatalln(err)
			}

			scope := cloud.NewScope(cloud.NewScopeParams{
				Cluster:       cluster,
				StoreProvider: storeProvider,
			})
			cm, err := scope.GetCloudManager()
			term.ExitOnError(err)

			kubeconfig, err := cm.GetKubeConfig()
			term.ExitOnError(err)

			err = cloud.UseCluster(opts, kubeconfig)
			if err != nil {
				term.Fatalln(err)
			}
		},
	}
	opts.AddFlags(cmd.Flags())
	return cmd
}
