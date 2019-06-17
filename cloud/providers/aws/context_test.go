package aws

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	cloudapi "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	"github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/store"
	_ "github.com/pharmer/pharmer/store/providers/fake"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func TestClusterManager_CreateCredentials(t *testing.T) {
	type fields struct {
		CloudManager *cloud.CloudManager
		conn         *cloudConnector
		namer        namer
	}
	type args struct {
		kc kubernetes.Interface
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "",
			fields: fields{
				CloudManager: &cloud.CloudManager{
					Cluster: &v1beta1.Cluster{
						TypeMeta:   v1.TypeMeta{},
						ObjectMeta: v1.ObjectMeta{},
						Spec: v1beta1.PharmerClusterSpec{
							ClusterAPI: v1alpha1.Cluster{},
							Config: v1beta1.ClusterConfig{
								MasterCount: 0,
								Cloud: v1beta1.CloudSpec{
									CloudProvider: "aws",
									Region:        "us-east-1",
								},
								CredentialName: "cred",
							},
						},
						Status: v1beta1.PharmerClusterStatus{},
					},
					Certs:       nil,
					AdminClient: nil,
					Credential:  nil,
				},
				conn:  nil,
				namer: namer{},
			},
			args: args{
				kc: fake.NewSimpleClientset(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := &ClusterManager{
				CloudManager: tt.fields.CloudManager,
				conn:         tt.fields.conn,
				namer:        tt.fields.namer,
			}
			var err error
			store.StoreProvider, err = store.NewStoreProvider(nil, "")
			if err != nil {
				t.Fatalf(err.Error())
			}

			createdSecret := &cloudapi.Credential{
				ObjectMeta: v1.ObjectMeta{
					Name: "cred",
				},
				Spec: cloudapi.CredentialSpec{
					Provider: "aws",
					Data: map[string]string{
						"accessKeyID":     "ABCDE",
						"secretAccessKey": "+abcd+efgh+",
					},
				},
			}

			_, err = store.StoreProvider.Credentials().Create(createdSecret)
			if err != nil {
				t.Error(err.Error())
			}

			if err := cm.CreateCredentials(tt.args.kc); (err != nil) != tt.wantErr {
				t.Errorf("ClusterManager.CreateCredentials() error = %v, wantErr %v", err, tt.wantErr)
			}

			secret, err := tt.args.kc.CoreV1().Secrets("aws-provider-system").Get("aws-provider-manager-bootstrap-credentials", v1.GetOptions{})
			if err != nil {
				t.Error(err.Error())
			}

			spew.Dump(string(secret.Data["credentials"]))
		})
	}
}
