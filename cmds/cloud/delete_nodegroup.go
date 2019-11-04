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
	"pharmer.dev/pharmer/cmds/cloud/options"
	"pharmer.dev/pharmer/store"

	"github.com/appscode/go/term"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewCmdDeleteNodeGroup() *cobra.Command {
	opts := options.NewNodeGroupDeleteConfig()
	cmd := &cobra.Command{
		Use: api.ResourceNameNodeGroup,
		Aliases: []string{
			api.ResourceTypeNodeGroup,
			api.ResourceCodeNodeGroup,
			api.ResourceKindNodeGroup,
		},
		Short:             "Delete a Kubernetes cluster NodeGroup",
		Example:           "pharmer delete nodegroup -k <cluster_name>",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.ValidateFlags(cmd, args); err != nil {
				term.Fatalln(err)
			}

			storeProvider, err := store.GetStoreProvider(cmd)
			term.ExitOnError(err)

			machinesetStore := storeProvider.MachineSet(opts.ClusterName)

			nodeGroups, err := getMachineSetList(machinesetStore, args...)
			term.ExitOnError(err)

			for _, ng := range nodeGroups {
				err := DeleteMachineSet(machinesetStore, ng.Name)
				term.ExitOnError(err)
			}
		},
	}
	opts.AddFlags(cmd.Flags())

	return cmd
}
func DeleteMachineSet(machinesetStore store.MachineSetStore, setName string) error {
	if setName == "" {
		return errors.New("missing machineset name")
	}

	mSet, err := machinesetStore.Get(setName)
	if err != nil {
		return errors.Errorf(`machinset not found in pharmer db, try using kubectl`)
	}
	tm := metav1.Now()
	mSet.DeletionTimestamp = &tm
	_, err = machinesetStore.Update(mSet)
	return err
}
