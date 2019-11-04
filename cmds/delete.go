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
	cpCmd "pharmer.dev/pharmer/cmds/cloud"
	"pharmer.dev/pharmer/cmds/credential"

	"github.com/spf13/cobra"
)

func newCmdDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "delete",
		DisableAutoGenTag: true,
		Run:               func(cmd *cobra.Command, args []string) {},
	}

	// Cloud
	cmd.AddCommand(cpCmd.NewCmdDeleteCluster())
	cmd.AddCommand(cpCmd.NewCmdDeleteNodeGroup())

	// Credential
	cmd.AddCommand(credential.NewCmdDeleteCredential())

	return cmd
}
