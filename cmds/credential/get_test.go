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
	"bytes"
	"testing"

	v1 "pharmer.dev/cloud/pkg/apis/cloud/v1"
	options2 "pharmer.dev/pharmer/cmds/credential/options"
	"pharmer.dev/pharmer/store"
	"pharmer.dev/pharmer/store/providers/fake"

	"github.com/onsi/gomega"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRunGetCredential(t *testing.T) {
	type args struct {
		credStore store.CredentialStore
		opts      *options2.CredentialGetConfig
	}
	tests := []struct {
		name       string
		args       args
		wantOut    string
		wantErr    bool
		beforeTest func(t *testing.T, a args) func(t *testing.T)
	}{
		{
			name: "doesn't exist",
			args: args{
				credStore: fake.New().Credentials(),
				opts: &options2.CredentialGetConfig{
					Credentials: []string{"aws"},
				},
			},
			wantOut: "",
			wantErr: true,
		},
		{
			name: "aws",
			args: args{
				credStore: fake.New().Credentials(),
				opts: &options2.CredentialGetConfig{
					Credentials: []string{"aws-cred"},
					Output:      "json",
				},
			},
			wantOut: "",
			wantErr: false,
			beforeTest: func(t *testing.T, a args) func(t *testing.T) {
				g := gomega.NewGomegaWithT(t)
				_, err := a.credStore.Create(&v1.Credential{
					ObjectMeta: v12.ObjectMeta{
						Name: "aws-cred",
					},
					Spec: v1.CredentialSpec{
						Provider: "aws",
						Data: map[string]string{
							"accessKeyID":     "a",
							"secretAccessKey": "b",
						},
					},
				})
				g.Expect(err).NotTo(gomega.HaveOccurred())

				return func(t *testing.T) {
					err = a.credStore.Delete("aws-cred")
					g.Expect(err).NotTo(gomega.HaveOccurred())
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.beforeTest != nil {
				afterTest := tt.beforeTest(t, tt.args)
				defer afterTest(t)
			}

			out := &bytes.Buffer{}
			if err := RunGetCredential(tt.args.credStore, tt.args.opts, out); (err != nil) != tt.wantErr {
				t.Errorf("RunGetCredential() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			//if gotOut := out.String(); gotOut != tt.wantOut {
			//	t.Errorf("RunGetCredential() = %v, want %v", gotOut, tt.wantOut)
			//}
		})
	}
}
