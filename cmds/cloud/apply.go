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
	"pharmer.dev/pharmer/cloud/utils/certificates"
	"pharmer.dev/pharmer/cmds/cloud/options"
	"pharmer.dev/pharmer/store"

	"github.com/appscode/go/term"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/klog/klogr"
)

func NewCmdApply() *cobra.Command {
	opts := options.NewApplyConfig()
	cmd := &cobra.Command{
		Use:               "apply",
		Short:             "Apply changes",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.ValidateFlags(cmd, args); err != nil {
				term.Fatalln(err)
			}

			storeProvider, err := store.GetStoreProvider(cmd)
			if err != nil {
				term.Fatalln(err)
			}

			err = runApplyCmd(storeProvider, opts)
			term.ExitOnError(err)

		},
	}
	opts.AddFlags(cmd.Flags())
	return cmd
}

func runApplyCmd(storeProvider store.ResourceInterface, opts *options.ApplyConfig) error {
	if opts.ClusterName == "" {
		return errors.New("missing Cluster name")
	}

	cluster, err := storeProvider.Clusters().Get(opts.ClusterName)
	if err != nil {
		return errors.Wrapf(err, "Cluster `%s` does not exist", opts.ClusterName)
	}
	certs, err := certificates.GetPharmerCerts(storeProvider, cluster.Name)
	if err != nil {
		return errors.Wrapf(err, "failed to get certs")
	}

	scope := cloud.NewScope(cloud.NewScopeParams{
		Cluster:       cluster,
		Certs:         certs,
		StoreProvider: storeProvider,
		Logger:        klogr.New().WithValues("cluster-name", cluster.Name),
	})

	return cloud.Apply(scope)
}
