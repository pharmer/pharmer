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
package credential

import (
	cloudapi "pharmer.dev/cloud/apis/cloud/v1"
	"pharmer.dev/pharmer/cmds/credential/options"
	"pharmer.dev/pharmer/store"

	"github.com/appscode/go/term"
	"github.com/spf13/cobra"
)

func NewCmdDeleteCredential() *cobra.Command {
	opts := options.NewCredentialDeleteConfig()
	cmd := &cobra.Command{
		Use: cloudapi.ResourceNameCredential,
		Aliases: []string{
			cloudapi.ResourceTypeCredential,
			cloudapi.ResourceCodeCredential,
			cloudapi.ResourceKindCredential,
		},
		Short:             "Delete  credential object",
		Example:           `pharmer delete credential`,
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.ValidateFlags(cmd, args); err != nil {
				term.Fatalln(err)
			}

			storeProvider, err := store.GetStoreProvider(cmd)
			term.ExitOnError(err)

			err = runDeleteCredentialCmd(storeProvider.Credentials(), opts)
			term.ExitOnError(err)
		},
	}
	opts.AddFlags(cmd.Flags())

	return cmd
}
func runDeleteCredentialCmd(credStore store.CredentialStore, opts *options.CredentialDeleteConfig) error {
	for _, cred := range opts.Credentials {
		err := credStore.Delete(cred)
		if err != nil {
			return err
		}
	}
	return nil
}
