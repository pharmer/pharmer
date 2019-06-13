package cloud_test

import (
	"testing"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/cloud/cmds/options"
	"github.com/pharmer/pharmer/cloud/providers/gce"

	_ "github.com/pharmer/pharmer/cloud/providers/aws"
	_ "github.com/pharmer/pharmer/cloud/providers/gce"
	"github.com/pharmer/pharmer/store"
	_ "github.com/pharmer/pharmer/store/providers/fake"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func getGCECluster() *api.Cluster {
	return &api.Cluster{
		ObjectMeta: v1.ObjectMeta{
			Name: "gce-cluster",
		},
		Spec: api.PharmerClusterSpec{
			ClusterAPI: clusterapi.Cluster{},
			Config: api.ClusterConfig{
				MasterCount: 3,
				Cloud: api.CloudSpec{
					CloudProvider: "gce",
					Zone:          "us-central-1f",
				},
				KubernetesVersion: "v1.14.0",
				CredentialName:    "gce-cred",
			},
		},
	}
}

func beforeTestCreate(t *testing.T) store.ResourceInterface {
	// create cluster
	storage, err := store.NewStoreProvider(nil, "")
	if err != nil {
		t.Fatalf(err.Error())
	}

	_, err = storage.Clusters().Create(&api.Cluster{
		ObjectMeta: v1.ObjectMeta{
			Name: "already-exists",
		},
	})
	if err != nil {
		t.Fatalf(err.Error())
	}

	return storage
}

func afterTestCreate(t *testing.T, store store.ResourceInterface) {
	// remove saved object
	err := store.Clusters().Delete("already-exists")
	if err != nil {
		t.Fatalf(err.Error())
	}
}

func checkFilesCreated(t *testing.T, store store.ResourceInterface, cluster *api.Cluster) {
	if store == nil {
		return
	}

	var allErr []error
	// check if cluster is created
	err := store.Clusters().Delete(cluster.Name)
	allErr = append(allErr, err)

	// check certificates are generated correctly
	err = store.Certificates(cluster.Name).Delete(api.CACertName)
	allErr = append(allErr, err)
	err = store.Certificates(cluster.Name).Delete(api.FrontProxyCACertName)
	allErr = append(allErr, err)
	err = store.Certificates(cluster.Name).Delete(api.ETCDCACertName)
	allErr = append(allErr, err)
	err = store.Certificates(cluster.Name).Delete(api.SAKeyName)
	allErr = append(allErr, err)
	err = store.SSHKeys(cluster.Name).Delete(cluster.GenSSHKeyExternalID())
	allErr = append(allErr, err)

	// check master machines
	for i := 0; i < cluster.Spec.Config.MasterCount; i++ {
		err = store.Machine(cluster.Name).Delete(cluster.MasterMachineName(i))
		allErr = append(allErr, err)
	}

	for _, err = range allErr {
		if err != nil {
			t.Error(err)
		}
	}
}

func TestCreate(t *testing.T) {
	storage := beforeTestCreate(t)
	defer afterTestCreate(t, storage)

	type args struct {
		store   store.ResourceInterface
		cluster *api.Cluster
	}
	tests := []struct {
		name    string
		args    args
		want    cloud.Interface
		wantErr bool
	}{

		{
			name: "nil cluster",
			args: args{
				cluster: nil,
			},
			want:    nil,
			wantErr: true,
		}, {
			name: "empty cluster-name",
			args: args{
				cluster: &api.Cluster{},
			},
			want:    nil,
			wantErr: true,
		}, {
			name: "empty kubernetes version",
			args: args{
				cluster: &api.Cluster{
					ObjectMeta: v1.ObjectMeta{
						Name: "test",
					},
				},
			},
			want:    nil,
			wantErr: true,
		}, {
			name: "cluster already exists",
			args: args{
				store: nil,
				cluster: &api.Cluster{
					ObjectMeta: v1.ObjectMeta{
						Name: "already-exists",
					},
				},
			},
			want:    nil,
			wantErr: true,
		}, {
			name: "gce cluster",
			args: args{
				store:   storage,
				cluster: getGCECluster(),
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "aws cluster",
			args: args{
				store: storage,
				cluster: &api.Cluster{
					ObjectMeta: v1.ObjectMeta{
						Name: "aws-cluster",
					},
					Spec: api.PharmerClusterSpec{
						ClusterAPI: clusterapi.Cluster{},
						Config: api.ClusterConfig{
							MasterCount: 3,
							Cloud: api.CloudSpec{
								CloudProvider: "aws",
								Zone:          "us-east-1b",
							},
							KubernetesVersion: "v1.14.0",
							CredentialName:    "aws-cred",
						},
					},
				},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "azure cluster",
			args: args{
				store: storage,
				cluster: &api.Cluster{
					ObjectMeta: v1.ObjectMeta{
						Name: "azure-cluster",
					},
					Spec: api.PharmerClusterSpec{
						ClusterAPI: clusterapi.Cluster{},
						Config: api.ClusterConfig{
							MasterCount: 3,
							Cloud: api.CloudSpec{
								CloudProvider: "azure",
								Zone:          "us-east-1b",
							},
							KubernetesVersion: "v1.14.0",
							CredentialName:    "azure-cred",
						},
					},
				},
			},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := cloud.Create(tt.args.store, tt.args.cluster)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			//if !reflect.DeepEqual(got, tt.want) {
			//	t.Errorf("Create() = %v, want %v", got, tt.want)
			//}

			// check all the files are created successfully
			checkFilesCreated(t, tt.args.store, tt.args.cluster)
		})
	}
}

func beforeTestCreateMachineSets(t *testing.T) store.ResourceInterface {
	localStore, err := store.NewStoreProvider(nil, "")
	if err != nil {
		t.Fatalf(err.Error())
	}

	return localStore
}

func checkMachinesetCreated(t *testing.T, store store.MachineSetStore, nodes map[string]int) {
	var allerr []error
	for node := range nodes {
		allerr = append(allerr, store.Delete(cloud.GenerateMachineSetName(node)))
	}
	for _, err := range allerr {
		if err != nil {
			t.Error(err)
		}
	}
}

func TestCreateMachineSets(t *testing.T) {
	localStore := beforeTestCreateMachineSets(t)

	type args struct {
		store store.ResourceInterface
		cm    cloud.Interface
		opts  *options.NodeGroupCreateConfig
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "no nodes",
			args: args{
				store: localStore,
				cm: &gce.ClusterManager{
					CloudManager: &cloud.CloudManager{
						Cluster: getGCECluster(),
					},
				},
				opts: &options.NodeGroupCreateConfig{
					Nodes: nil,
				},
			},
			wantErr: false,
		}, {
			name: "test gce",
			args: args{
				store: localStore,
				cm: &gce.ClusterManager{
					CloudManager: &cloud.CloudManager{
						Cluster: getGCECluster(),
					},
				},
				opts: &options.NodeGroupCreateConfig{
					Nodes: map[string]int{
						"a": 1,
						"b": 2,
						"c": 3,
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := cloud.CreateMachineSets(tt.args.store, tt.args.cm, tt.args.opts); (err != nil) != tt.wantErr {
				t.Errorf("CreateMachineSets() error = %v, wantErr %v", err, tt.wantErr)
			}
			checkMachinesetCreated(t, tt.args.store.MachineSet(tt.args.cm.GetCluster().Name), tt.args.opts.Nodes)
		})
	}
}
