package credential

import (
	"bytes"
	"testing"

	options2 "github.com/pharmer/pharmer/cmds/credential/options"
	"github.com/pharmer/pharmer/store"
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
