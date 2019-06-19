package cloud_test

import (
	"testing"

	"github.com/onsi/gomega"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/cloud/cmds/options"
	_ "github.com/pharmer/pharmer/cloud/providers/aws"
	_ "github.com/pharmer/pharmer/cloud/providers/gce"
	"github.com/pharmer/pharmer/cloud/utils/certificates"
	"github.com/pharmer/pharmer/store"
	"github.com/pharmer/pharmer/store/providers/fake"
	_ "github.com/pharmer/pharmer/store/providers/fake"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

//_, err = storage.Credentials().Create(&cloudapi.Credential{
//	ObjectMeta: v1.ObjectMeta{
//		Name: "azure-cred",
//	},
//	Spec: cloudapi.CredentialSpec{
//		Provider: "azure",
//		Data: map[string]string{
//			"clientID":       "a",
//			"clientSecret":   "b",
//			"subscriptionID": "c",
//			"tenantID":       "d",
//		},
//	},
//})

func deleteCerts(t *testing.T, s store.ResourceInterface, clusterName string) {
	t.Helper()
	g := gomega.NewGomegaWithT(t)
	err := s.Certificates(clusterName).Delete(api.CACertName)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	err = s.Certificates(clusterName).Delete(api.FrontProxyCACertName)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	err = s.Certificates(clusterName).Delete(api.ETCDCACertName)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	err = s.Certificates(clusterName).Delete(api.SAKeyName)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	err = s.SSHKeys(clusterName).Delete(clusterName + "-sshkey")
	g.Expect(err).NotTo(gomega.HaveOccurred())
}

func TestCreateCluster(t *testing.T) {
	type args struct {
		store   store.ResourceInterface
		cluster *api.Cluster
	}

	genericBeforeTest := func(t *testing.T, a args) func(*testing.T) {
		g := gomega.NewGomegaWithT(t)
		return func(t *testing.T) {
			// check if Cluster is created
			err := a.store.Clusters().Delete(a.cluster.Name)
			g.Expect(err).NotTo(gomega.HaveOccurred())

			deleteCerts(t, a.store, a.cluster.Name)

			// check master machines are genereated
			for i := 0; i < a.cluster.Spec.Config.MasterCount; i++ {
				err = a.store.Machine(a.cluster.Name).Delete(a.cluster.MasterMachineName(i))
				g.Expect(err).NotTo(gomega.HaveOccurred())
			}
		}
	}

	tests := []struct {
		name       string
		args       args
		wantErr    bool
		errmsg     string
		beforeTest func(*testing.T, args) func(*testing.T)
	}{
		{
			name: "nil Cluster",
			args: args{
				cluster: nil,
			},
			wantErr: true,
			errmsg:  "missing Cluster",
		}, {
			name: "empty Cluster-name",
			args: args{
				cluster: &api.Cluster{},
			},
			wantErr: true,
			errmsg:  "missing Cluster name",
		}, {
			name: "empty kubernetes version",
			args: args{
				cluster: &api.Cluster{
					ObjectMeta: v1.ObjectMeta{
						Name: "test",
					},
				},
			},
			wantErr: true,
			errmsg:  "missing Cluster version",
		}, {
			name: "Cluster already exists",
			args: args{
				store: fake.New(),
				cluster: &api.Cluster{
					ObjectMeta: v1.ObjectMeta{
						Name: "already-exists",
					},
					Spec: api.PharmerClusterSpec{
						Config: api.ClusterConfig{
							KubernetesVersion: "1.13.0",
						},
					},
				},
			},
			wantErr: true,
			errmsg:  "cluster already exists",
			beforeTest: func(t *testing.T, a args) func(t *testing.T) {
				_, err := a.store.Clusters().Create(&api.Cluster{
					ObjectMeta: v1.ObjectMeta{
						Name: "already-exists",
					},
				})
				if err != nil {
					t.Errorf("failed to create Cluster: %v", err)
				}

				return func(t *testing.T) {
					err = a.store.Clusters().Delete("already-exists")
					if err != nil {
						t.Errorf("failed to delete Cluster: %v", err)
					}
				}
			},
		}, {
			name: "gce Cluster",
			args: args{
				store: fake.New(),
				cluster: &api.Cluster{
					ObjectMeta: v1.ObjectMeta{
						Name: "gce",
					},
					Spec: api.PharmerClusterSpec{
						ClusterAPI: v1alpha1.Cluster{},
						Config: api.ClusterConfig{
							MasterCount: 3,
							Cloud: api.CloudSpec{
								CloudProvider: "gce",
								Zone:          "us-central-1f",
							},
							KubernetesVersion: "1.13.1",
						},
					},
				},
			},
			wantErr:    false,
			beforeTest: genericBeforeTest,
		}, {
			name: "aws Cluster",
			args: args{
				store: fake.New(),
				cluster: &api.Cluster{
					ObjectMeta: v1.ObjectMeta{
						Name: "aws",
					},
					Spec: api.PharmerClusterSpec{
						ClusterAPI: v1alpha1.Cluster{},
						Config: api.ClusterConfig{
							MasterCount: 3,
							Cloud: api.CloudSpec{
								CloudProvider: "aws",
								Zone:          "us-east-1b",
							},
							KubernetesVersion: "1.13.1",
						},
					},
				},
			},
			wantErr:    false,
			beforeTest: genericBeforeTest,
		},
	}
	g := gomega.NewGomegaWithT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.beforeTest != nil {
				afterTest := tt.beforeTest(t, tt.args)
				defer afterTest(t)
			}

			err := cloud.CreateCluster(tt.args.store, tt.args.cluster)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateCluster() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				g.Expect(err).Should(gomega.MatchError(tt.errmsg))
			}
		})
	}
}

func TestCreateMachineSets(t *testing.T) {
	type args struct {
		store store.ResourceInterface
		opts  *options.NodeGroupCreateConfig
	}
	tests := []struct {
		name       string
		args       args
		wantErr    bool
		beforeTest func(*testing.T, args) func(*testing.T)
	}{
		{
			name: "no nodes",
			args: args{
				store: fake.New(),
				opts: &options.NodeGroupCreateConfig{
					ClusterName: "gce",
					Nodes:       nil,
				},
			},
			wantErr: false,
			beforeTest: func(t *testing.T, a args) func(*testing.T) {
				cluster, err := a.store.Clusters().Create(&api.Cluster{
					ObjectMeta: v1.ObjectMeta{
						Name: a.opts.ClusterName,
					},
				})
				if err != nil {
					t.Errorf("failed to create Cluster: %v", err)
				}

				return func(t *testing.T) {
					err = a.store.Clusters().Delete(cluster.Name)
					if err != nil {
						t.Errorf("failed to delete Cluster: %v", err)
					}
				}
			},
		},
		{
			name: "test gce",
			args: args{
				store: fake.New(),
				opts: &options.NodeGroupCreateConfig{
					ClusterName: "gce",
					Nodes: map[string]int{
						"a": 1,
						"b": 2,
						"c": 3,
					},
				},
			},
			wantErr: false,
			beforeTest: func(t *testing.T, a args) func(*testing.T) {
				cluster, err := a.store.Clusters().Create(&api.Cluster{
					ObjectMeta: v1.ObjectMeta{
						Name: a.opts.ClusterName,
					},
					Spec: api.PharmerClusterSpec{
						Config: api.ClusterConfig{
							Cloud: api.CloudSpec{
								CloudProvider: "gce",
								Zone:          "us-central-1f",
							},
							KubernetesVersion: "1.13.4",
						},
					},
				})
				g := gomega.NewGomegaWithT(t)
				g.Expect(err).NotTo(gomega.HaveOccurred())

				_, err = certificates.CreateCertsKeys(a.store, a.opts.ClusterName)
				g.Expect(err).NotTo(gomega.HaveOccurred())

				return func(t *testing.T) {
					err = a.store.Clusters().Delete(cluster.Name)
					g.Expect(err).NotTo(gomega.HaveOccurred())

					deleteCerts(t, a.store, a.opts.ClusterName)

					// check if machinesets are created
					for node := range a.opts.Nodes {
						err = a.store.MachineSet(a.opts.ClusterName).Delete(cloud.GenerateMachineSetName(node))
						g.Expect(err).NotTo(gomega.HaveOccurred())
					}
				}
			},
		},
		{
			name: "test aws",
			args: args{
				store: fake.New(),
				opts: &options.NodeGroupCreateConfig{
					ClusterName: "aws",
					Nodes: map[string]int{
						"a": 1,
						"b": 2,
						"c": 3,
					},
				},
			},
			wantErr: false,
			beforeTest: func(t *testing.T, a args) func(*testing.T) {
				cluster, err := a.store.Clusters().Create(&api.Cluster{
					ObjectMeta: v1.ObjectMeta{
						Name: a.opts.ClusterName,
					},
					Spec: api.PharmerClusterSpec{
						Config: api.ClusterConfig{
							Cloud: api.CloudSpec{
								CloudProvider: "aws",
								Zone:          "us-east-1b",
								AWS: &api.AWSSpec{
									IAMProfileMaster: "master",
									IAMProfileNode:   "node",
								},
							},
							KubernetesVersion: "1.13.4",
						},
					},
				})
				g := gomega.NewGomegaWithT(t)
				g.Expect(err).NotTo(gomega.HaveOccurred())

				_, err = certificates.CreateCertsKeys(a.store, a.opts.ClusterName)
				g.Expect(err).NotTo(gomega.HaveOccurred())

				return func(t *testing.T) {
					err = a.store.Clusters().Delete(cluster.Name)
					g.Expect(err).NotTo(gomega.HaveOccurred())

					deleteCerts(t, a.store, a.opts.ClusterName)

					// check if machinesets are created
					for node := range a.opts.Nodes {
						err = a.store.MachineSet(a.opts.ClusterName).Delete(cloud.GenerateMachineSetName(node))
						g.Expect(err).NotTo(gomega.HaveOccurred())
					}
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			afterTest := tt.beforeTest(t, tt.args)
			defer afterTest(t)

			if err := cloud.CreateMachineSets(tt.args.store, tt.args.opts); (err != nil) != tt.wantErr {
				t.Errorf("CreateMachineSets() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
