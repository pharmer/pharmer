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
	api "pharmer.dev/pharmer/apis/v1alpha1"
	"pharmer.dev/pharmer/cloud"
	"pharmer.dev/pharmer/cmds/cloud/options"
	"pharmer.dev/pharmer/store"

	"github.com/appscode/go/term"
	"github.com/spf13/cobra"
	"k8s.io/klog/klogr"
)

func NewCmdCreateCluster() *cobra.Command {
	opts := options.NewClusterCreateConfig()
	cmd := &cobra.Command{
		Use: api.ResourceNameCluster,
		Aliases: []string{
			api.ResourceTypeCluster,
			api.ResourceKindCluster,
		},
		Short:             "Create a Kubernetes cluster for a given cloud provider",
		Example:           "pharmer create cluster demo-cluster",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.ValidateFlags(cmd, args); err != nil {
				term.Fatalln(err)
			}

			storeProvider, err := store.GetStoreProvider(cmd)
			if err != nil {
				term.Fatalln(err)
			}

			err = runCreateClusterCmd(storeProvider, opts)
			if err != nil {
				term.Fatalln(err)
			}
		},
	}
	opts.AddFlags(cmd.Flags())

	return cmd
}

func runCreateClusterCmd(store store.ResourceInterface, opts *options.ClusterCreateConfig) error {
	scope := cloud.NewScope(cloud.NewScopeParams{
		Cluster:       opts.Cluster,
		StoreProvider: store,
		Logger:        klogr.New().WithValues("cluster-name", opts.Cluster.Name),
	})

	err := cloud.CreateCluster(scope)
	if err != nil {
		return err
	}

	if len(opts.Nodes) > 0 {
		nodeOpts := options.NewNodeGroupCreateConfig()
		nodeOpts.ClusterName = opts.Cluster.Name
		nodeOpts.Nodes = opts.Nodes
		err := cloud.CreateMachineSets(store, nodeOpts)
		if err != nil {
			return err
		}
	}

	return nil
}
