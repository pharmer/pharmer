package credential

import (
	"os"
	"testing"

	options2 "github.com/pharmer/pharmer/cmds/credential/options"
	"github.com/pharmer/pharmer/store"
	"github.com/pharmer/pharmer/store/providers/fake"
)

// TODO:
// write tests:
// create credential
// delete credential
// get credential
// edit credential
// for all providers, with all available flags

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
					FromFile: "",
					Issue:    false,
					Owner:    "",
				},
			},
			wantErr: false,
			beforeTest: func(t *testing.T, a args) func(*testing.T) {
				os.Setenv("DIGITALOCEAN_TOKEN", "abcd")
				return func(t *testing.T) {
					os.Unsetenv("DIGITALOCEAN_TOKEN")
				}
			},
		},
		// TODO: add more
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := runCreateCredential(tt.args.credentialStore, tt.args.opts); (err != nil) != tt.wantErr {
				t.Errorf("runCreateCredential() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
