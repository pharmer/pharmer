package cloud_test

import (
	"reflect"
	"testing"

	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/cloud/providers/aws"
	"github.com/pharmer/pharmer/cloud/utils/certificates"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func TestNewNodeTemplateData(t *testing.T) {
	type args struct {
		cm      cloud.Interface
		machine *clusterv1.Machine
		token   string
	}
	var tests = []struct {
		name string
		args args
		want cloud.TemplateData
	}{
		{
			name: "",
			args: args{
				cm: &aws.ClusterManager{
					CloudManager: &cloud.CloudManager{
						Cluster:     nil,
						Certs:       &certificates.Certificates{},
						AdminClient: nil,
						Credential:  nil,
					},
				},
				machine: nil,
				token:   "",
			},
			want: cloud.TemplateData{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cloud.NewNodeTemplateData(tt.args.cm, tt.args.machine, tt.args.token); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewNodeTemplateData() = %v, want %v", got, tt.want)
			}
		})
	}
}
