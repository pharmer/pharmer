package credential

import (
	"testing"

	"github.com/pharmer/pharmer/cmds/credential/options"
	"github.com/pharmer/pharmer/store"
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
