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
	"os"
	"testing"

	options2 "pharmer.dev/pharmer/cmds/credential/options"
	"pharmer.dev/pharmer/store"
	"pharmer.dev/pharmer/store/providers/fake"
)

// TODO:
// write tests:
// create credential
// delete credential
// get credential
// edit credential
// for all providers, with all available flags

// TODO:
// only checking if cred is created
// should we test file contents?

func Test_runCreateCredential(t *testing.T) {
	type args struct {
		credentialStore store.CredentialStore
		opts            *options2.CredentialCreateConfig
	}
	tests := []struct {
		name       string
		args       args
		wantErr    bool
		beforeTest func(*testing.T, args) func(*testing.T)
	}{
		{
			name: "digitalocean from env",
			args: args{
				credentialStore: fake.New().Credentials(),
				opts: &options2.CredentialCreateConfig{
					Name:     "do-test",
					Provider: "digitalocean",
					FromEnv:  true,
				},
			},
			wantErr: false,
			beforeTest: func(t *testing.T, a args) func(*testing.T) {
				os.Setenv("DIGITALOCEAN_TOKEN", "abcd")
				return func(t *testing.T) {
					os.Unsetenv("DIGITALOCEAN_TOKEN")
					err := a.credentialStore.Delete(a.opts.Name)
					if err != nil {
						t.Errorf("failed to delete cred")
					}
				}
			},
		},
		{
			name: "aws from env",
			args: args{
				credentialStore: fake.New().Credentials(),
				opts: &options2.CredentialCreateConfig{
					Name:     "aws-test",
					Provider: "aws",
					FromEnv:  true,
				},
			},
			wantErr: false,
			beforeTest: func(t *testing.T, a args) func(*testing.T) {
				os.Setenv("AWS_ACCESS_KEY_ID", "abcd")
				os.Setenv("AWS_ACCESS_SECRET_ACCESS_KEY", "abcd")
				return func(t *testing.T) {
					os.Unsetenv("AWS_ACCESS_KEY_ID")
					os.Unsetenv("AWS_ACCESS_SECRET_ACCESS_KEY")

					err := a.credentialStore.Delete(a.opts.Name)
					if err != nil {
						t.Errorf("failed to delete cred")
					}
				}
			},
		},
		{
			name: "azure from env",
			args: args{
				credentialStore: fake.New().Credentials(),
				opts: &options2.CredentialCreateConfig{
					Name:     "azure-test",
					Provider: "azure",
					FromEnv:  true,
				},
			},
			wantErr: false,
			beforeTest: func(t *testing.T, a args) func(*testing.T) {
				os.Setenv("AZURE_SUBSCRIPTION_ID", "a")
				os.Setenv("AZURE_TENANT_ID", "b")
				os.Setenv("AZURE_CLIENT_ID", "c")
				os.Setenv("AZURE_CLIENT_SECRET", "d")

				return func(t *testing.T) {
					os.Unsetenv("AZURE_SUBSCRIPTION_ID")
					os.Unsetenv("AZURE_TENANT_ID")
					os.Unsetenv("AZURE_CLIENT_ID")
					os.Unsetenv("AZURE_CLIENT_SECRET")

					err := a.credentialStore.Delete(a.opts.Name)
					if err != nil {
						t.Errorf("failed to delete cred")
					}
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := runCreateCredential(tt.args.credentialStore, tt.args.opts); (err != nil) != tt.wantErr {
				t.Errorf("runCreateCredential() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
