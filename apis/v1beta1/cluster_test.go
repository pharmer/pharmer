package v1beta1

import (
	"reflect"
	"testing"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func TestCluster_ClusterConfig(t *testing.T) {
	type fields struct {
		TypeMeta   metav1.TypeMeta
		ObjectMeta metav1.ObjectMeta
		Spec       PharmerClusterSpec
		Status     PharmerClusterStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   ClusterConfig
	}{
		{
			name: "test cluster config",
			fields: fields{
				Spec: PharmerClusterSpec{
					Config: ClusterConfig{
						MasterCount:       3,
						KubernetesVersion: "1.13.5",
					},
				},
			},
			want: ClusterConfig{
				MasterCount:       3,
				KubernetesVersion: "1.13.5",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cluster{
				TypeMeta:   tt.fields.TypeMeta,
				ObjectMeta: tt.fields.ObjectMeta,
				Spec:       tt.fields.Spec,
				Status:     tt.fields.Status,
			}
			if got := c.ClusterConfig(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Cluster.ClusterConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCluster_APIServerURL(t *testing.T) {
	type fields struct {
		TypeMeta   metav1.TypeMeta
		ObjectMeta metav1.ObjectMeta
		Spec       PharmerClusterSpec
		Status     PharmerClusterStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "nil adresses",
			fields: fields{
				Spec: PharmerClusterSpec{
					ClusterAPI: v1alpha1.Cluster{
						Status: v1alpha1.ClusterStatus{
							APIEndpoints: nil,
						},
					},
				},
			},
			want: "",
		},
		{
			name: "port 0",
			fields: fields{
				Spec: PharmerClusterSpec{
					ClusterAPI: v1alpha1.Cluster{
						Status: v1alpha1.ClusterStatus{
							APIEndpoints: []v1alpha1.APIEndpoint{
								{
									Host: "1.2.3.4",
									Port: 0,
								},
							},
						},
					},
				},
			},
			want: "https://1.2.3.4",
		},
		{
			name: "port not 0",
			fields: fields{
				Spec: PharmerClusterSpec{
					ClusterAPI: v1alpha1.Cluster{
						Status: v1alpha1.ClusterStatus{
							APIEndpoints: []v1alpha1.APIEndpoint{
								{
									Host: "1.2.3.4",
									Port: 6443,
								},
							},
						},
					},
				},
			},
			want: "https://1.2.3.4:6443",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cluster{
				TypeMeta:   tt.fields.TypeMeta,
				ObjectMeta: tt.fields.ObjectMeta,
				Spec:       tt.fields.Spec,
				Status:     tt.fields.Status,
			}
			if got := c.APIServerURL(); got != tt.want {
				t.Errorf("Cluster.APIServerURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCluster_SetClusterAPIEndpoints(t *testing.T) {
	type fields struct {
		TypeMeta   metav1.TypeMeta
		ObjectMeta metav1.ObjectMeta
		Spec       PharmerClusterSpec
		Status     PharmerClusterStatus
	}
	type args struct {
		addresses []core.NodeAddress
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "set api-endpoint",
			args: args{
				addresses: []core.NodeAddress{
					{
						Type:    core.NodeExternalIP,
						Address: "1.2.3.4",
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cluster{
				TypeMeta:   tt.fields.TypeMeta,
				ObjectMeta: tt.fields.ObjectMeta,
				Spec:       tt.fields.Spec,
				Status:     tt.fields.Status,
			}
			if err := c.SetClusterAPIEndpoints(tt.args.addresses); (err != nil) != tt.wantErr {
				t.Errorf("Cluster.SetClusterAPIEndpoints() error = %v, wantErr %v", err, tt.wantErr)
			}
			apiEndpoints := c.Spec.ClusterAPI.Status.APIEndpoints
			newEndPoint := v1alpha1.APIEndpoint{
				Host: tt.args.addresses[0].Address,
				Port: 6443,
			}
			if apiEndpoints[len(apiEndpoints)-1] != newEndPoint {
				t.Errorf("apiendpoint not set")
			}
		})
	}
}

func TestCluster_APIServerAddress(t *testing.T) {
	type fields struct {
		TypeMeta   metav1.TypeMeta
		ObjectMeta metav1.ObjectMeta
		Spec       PharmerClusterSpec
		Status     PharmerClusterStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "port set",
			fields: fields{
				Spec: PharmerClusterSpec{
					ClusterAPI: v1alpha1.Cluster{
						Status: v1alpha1.ClusterStatus{
							APIEndpoints: []v1alpha1.APIEndpoint{
								{
									Host: "1.2.3.4",
									Port: 6443,
								},
							},
						},
					},
				},
			},
			want: "1.2.3.4:6443",
		},
		{
			name: "port not set",
			fields: fields{
				Spec: PharmerClusterSpec{
					ClusterAPI: v1alpha1.Cluster{
						Status: v1alpha1.ClusterStatus{
							APIEndpoints: []v1alpha1.APIEndpoint{
								{
									Host: "1.2.3.4",
								},
							},
						},
					},
				},
			},
			want: "1.2.3.4",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cluster{
				TypeMeta:   tt.fields.TypeMeta,
				ObjectMeta: tt.fields.ObjectMeta,
				Spec:       tt.fields.Spec,
				Status:     tt.fields.Status,
			}
			if got := c.APIServerAddress(); got != tt.want {
				t.Errorf("Cluster.APIServerAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCluster_SetNetworkingDefaults(t *testing.T) {
	type fields struct {
		TypeMeta   metav1.TypeMeta
		ObjectMeta metav1.ObjectMeta
		Spec       PharmerClusterSpec
		Status     PharmerClusterStatus
	}
	type args struct {
		provider string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "",
			fields: fields{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{},
				Spec:       PharmerClusterSpec{},
				Status:     PharmerClusterStatus{},
			},
			args: args{
				provider: "calico",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cluster{
				TypeMeta:   tt.fields.TypeMeta,
				ObjectMeta: tt.fields.ObjectMeta,
				Spec:       tt.fields.Spec,
				Status:     tt.fields.Status,
			}
			c.SetNetworkingDefaults(tt.args.provider)
			if c.Spec.ClusterAPI.Spec.ClusterNetwork.Services.CIDRBlocks[0] != v1beta1.DefaultServicesSubnet {
				t.Errorf("service subnet not set")
			}
			if c.Spec.ClusterAPI.Spec.ClusterNetwork.ServiceDomain != v1beta1.DefaultServiceDNSDomain {
				t.Errorf("service domain not set")
			}
			if c.Spec.ClusterAPI.Spec.ClusterNetwork.Pods.CIDRBlocks[0] != "192.168.0.0/16" {
				t.Errorf("pod cidr not set")
			}
		})
	}
}

func TestCluster_InitClusterAPI(t *testing.T) {
	type fields struct {
		TypeMeta   metav1.TypeMeta
		ObjectMeta metav1.ObjectMeta
		Spec       PharmerClusterSpec
		Status     PharmerClusterStatus
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "",
			fields: fields{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-1",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cluster{
				TypeMeta:   tt.fields.TypeMeta,
				ObjectMeta: tt.fields.ObjectMeta,
				Spec:       tt.fields.Spec,
				Status:     tt.fields.Status,
			}
			c.InitClusterAPI()
			if c.Spec.ClusterAPI.Name != c.Name {
				t.Errorf("name not set")
			}
		})
	}
}

func TestCluster_IsLessThanVersion(t *testing.T) {
	type fields struct {
		TypeMeta   metav1.TypeMeta
		ObjectMeta metav1.ObjectMeta
		Spec       PharmerClusterSpec
		Status     PharmerClusterStatus
	}
	type args struct {
		in string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "same version",
			fields: fields{
				Spec: PharmerClusterSpec{
					Config: ClusterConfig{
						KubernetesVersion: "1.13.5",
					},
				},
			},
			args: args{
				in: "1.13.5",
			},
			want: false,
		},
		{
			name: "less patch versoin",
			fields: fields{
				Spec: PharmerClusterSpec{
					Config: ClusterConfig{
						KubernetesVersion: "1.13.6",
					},
				},
			},
			args: args{
				in: "1.13.5",
			},
			want: false,
		},
		{
			name: "less minor version",
			fields: fields{
				Spec: PharmerClusterSpec{
					Config: ClusterConfig{
						KubernetesVersion: "1.14.0",
					},
				},
			},
			args: args{
				in: "1.13.5",
			},
			want: false,
		},
		{
			name: "more minor version",
			fields: fields{
				Spec: PharmerClusterSpec{
					Config: ClusterConfig{
						KubernetesVersion: "1.13.5",
					},
				},
			},
			args: args{
				in: "1.14.0",
			},
			want: true,
		},
		{
			name: "more patch version",
			fields: fields{
				Spec: PharmerClusterSpec{
					Config: ClusterConfig{
						KubernetesVersion: "1.13.5",
					},
				},
			},
			args: args{
				in: "1.13.6",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Cluster{
				TypeMeta:   tt.fields.TypeMeta,
				ObjectMeta: tt.fields.ObjectMeta,
				Spec:       tt.fields.Spec,
				Status:     tt.fields.Status,
			}
			if got := c.IsLessThanVersion(tt.args.in); got != tt.want {
				t.Errorf("Cluster.IsLessThanVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCluster_GenSSHKeyExternalID(t *testing.T) {
	type fields struct {
		TypeMeta   metav1.TypeMeta
		ObjectMeta metav1.ObjectMeta
		Spec       PharmerClusterSpec
		Status     PharmerClusterStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "",
			fields: fields{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster",
				},
			},
			want: "test-cluster-sshkey",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cluster{
				TypeMeta:   tt.fields.TypeMeta,
				ObjectMeta: tt.fields.ObjectMeta,
				Spec:       tt.fields.Spec,
				Status:     tt.fields.Status,
			}
			if got := c.GenSSHKeyExternalID(); got != tt.want {
				t.Errorf("Cluster.GenSSHKeyExternalID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCluster_MasterMachineName(t *testing.T) {
	type fields struct {
		TypeMeta   metav1.TypeMeta
		ObjectMeta metav1.ObjectMeta
		Spec       PharmerClusterSpec
		Status     PharmerClusterStatus
	}
	type args struct {
		n int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "",
			fields: fields{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster",
				},
			},
			args: args{
				n: 0,
			},
			want: "test-cluster-master-0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cluster{
				TypeMeta:   tt.fields.TypeMeta,
				ObjectMeta: tt.fields.ObjectMeta,
				Spec:       tt.fields.Spec,
				Status:     tt.fields.Status,
			}
			if got := c.MasterMachineName(tt.args.n); got != tt.want {
				t.Errorf("Cluster.MasterMachineName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCluster_CloudProvider(t *testing.T) {
	type fields struct {
		TypeMeta   metav1.TypeMeta
		ObjectMeta metav1.ObjectMeta
		Spec       PharmerClusterSpec
		Status     PharmerClusterStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "",
			fields: fields{
				Spec: PharmerClusterSpec{
					Config: ClusterConfig{
						Cloud: CloudSpec{
							CloudProvider: "aws",
						},
					},
				},
			},
			want: "aws",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cluster{
				TypeMeta:   tt.fields.TypeMeta,
				ObjectMeta: tt.fields.ObjectMeta,
				Spec:       tt.fields.Spec,
				Status:     tt.fields.Status,
			}
			if got := c.CloudProvider(); got != tt.want {
				t.Errorf("Cluster.CloudProvider() = %v, want %v", got, tt.want)
			}
		})
	}
}
