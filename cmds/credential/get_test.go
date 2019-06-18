package credential

import (
	"bytes"
	"testing"

	options2 "github.com/pharmer/pharmer/cmds/credential/options"
	"github.com/pharmer/pharmer/store"
)

func TestRunGetCredential(t *testing.T) {
	type args struct {
		credStore store.CredentialStore
		opts      *options2.CredentialGetConfig
	}
	tests := []struct {
		name    string
		args    args
		wantOut string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			if err := RunGetCredential(tt.args.credStore, tt.args.opts, out); (err != nil) != tt.wantErr {
				t.Errorf("RunGetCredential() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotOut := out.String(); gotOut != tt.wantOut {
				t.Errorf("RunGetCredential() = %v, want %v", gotOut, tt.wantOut)
			}
		})
	}
}
