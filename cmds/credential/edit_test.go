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

	options2 "pharmer.dev/pharmer/cmds/credential/options"
	"pharmer.dev/pharmer/store"
)

func TestRunUpdateCredential(t *testing.T) {
	type args struct {
		credStore store.CredentialStore
		opts      *options2.CredentialEditConfig
	}
	tests := []struct {
		name       string
		args       args
		wantErrOut string
		wantErr    bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errOut := &bytes.Buffer{}
			if err := RunUpdateCredential(tt.args.credStore, tt.args.opts, errOut); (err != nil) != tt.wantErr {
				t.Errorf("RunUpdateCredential() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotErrOut := errOut.String(); gotErrOut != tt.wantErrOut {
				t.Errorf("RunUpdateCredential() = %v, want %v", gotErrOut, tt.wantErrOut)
			}
		})
	}
}
