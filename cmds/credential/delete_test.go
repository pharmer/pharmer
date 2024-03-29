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
	"testing"

	"pharmer.dev/pharmer/cmds/credential/options"
	"pharmer.dev/pharmer/store"
)

func Test_runDeleteCredentialCmd(t *testing.T) {
	type args struct {
		credStore store.CredentialStore
		opts      *options.CredentialDeleteConfig
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: add tests
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := runDeleteCredentialCmd(tt.args.credStore, tt.args.opts); (err != nil) != tt.wantErr {
				t.Errorf("runDeleteCredentialCmd() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
