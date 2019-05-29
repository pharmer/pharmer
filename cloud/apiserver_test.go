package cloud

import (
	"fmt"
	"testing"

	"github.com/pharmer/pharmer/store"

	"github.com/pharmer/pharmer/config"
	_ "github.com/pharmer/pharmer/store/providers/vfs"
	"github.com/pharmer/pharmer/utils"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCreateCredentialSecret(t *testing.T) {
	type args struct {
		client  kubernetes.Interface
		cluster *api.Cluster
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "tests-1",
			args: args{
				client: fake.NewSimpleClientset(),
				cluster: &api.Cluster{
					Spec: api.PharmerClusterSpec{
						Config: &api.ClusterConfig{
							Cloud: api.CloudSpec{
								CloudProvider: "gce",
							},
							CredentialName: "google",
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store.StoreProvider = store.NewStoreProvider(config.NewDefaultConfig(), utils.GetLocalOwner())
			fmt.Println(store.StoreProvider)
			if err := CreateCredentialSecret(tt.args.client, tt.args.cluster); (err != nil) != tt.wantErr {
				t.Errorf("CreateCredentialSecret() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
