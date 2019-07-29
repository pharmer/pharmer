package cloud_test

import (
	"testing"

	"github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/klogr"
	cloudapi "pharmer.dev/cloud/pkg/apis/cloud/v1"
	api "pharmer.dev/pharmer/apis/v1alpha1"
	"pharmer.dev/pharmer/cloud"
	_ "pharmer.dev/pharmer/cloud/providers/aks"
	_ "pharmer.dev/pharmer/cloud/providers/aws"
	"pharmer.dev/pharmer/cloud/providers/azure"
	_ "pharmer.dev/pharmer/cloud/providers/azure"
	_ "pharmer.dev/pharmer/cloud/providers/digitalocean"
	_ "pharmer.dev/pharmer/cloud/providers/dokube"
	_ "pharmer.dev/pharmer/cloud/providers/eks"
	_ "pharmer.dev/pharmer/cloud/providers/gce"
	_ "pharmer.dev/pharmer/cloud/providers/gke"
	_ "pharmer.dev/pharmer/cloud/providers/linode"
	_ "pharmer.dev/pharmer/cloud/providers/packet"
	"pharmer.dev/pharmer/cloud/utils/certificates"
	"pharmer.dev/pharmer/cmds/cloud/options"
	"pharmer.dev/pharmer/store"
	"pharmer.dev/pharmer/store/providers/fake"
	_ "pharmer.dev/pharmer/store/providers/fake"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

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
		scope *cloud.Scope
	}

	genericBeforeTest := func(t *testing.T, a args) func(*testing.T) {
		s := a.scope
		g := gomega.NewGomegaWithT(t)
		return func(t *testing.T) {
			// check if Cluster is created
			err := s.StoreProvider.Clusters().Delete(s.Cluster.Name)
			g.Expect(err).NotTo(gomega.HaveOccurred())

			deleteCerts(t, s.StoreProvider, s.Cluster.Name)

			if !api.ManagedProviders.Has(s.Cluster.CloudProvider()) {
				// check master machines are genereated
				for i := 0; i < s.Cluster.Spec.Config.MasterCount; i++ {
					err = s.StoreProvider.Machine(s.Cluster.Name).Delete(s.Cluster.MasterMachineName(i))
					g.Expect(err).NotTo(gomega.HaveOccurred())
				}
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
				scope: &cloud.Scope{
					Cluster: nil,
					Logger:  klogr.New(),
				},
			},
			wantErr: true,
			errmsg:  "missing Cluster",
		}, {
			name: "empty Cluster-name",
			args: args{
				scope: &cloud.Scope{
					Cluster: &api.Cluster{},
					Logger:  klogr.New(),
				},
			},
			wantErr: true,
			errmsg:  "missing Cluster name",
		}, {
			name: "empty kubernetes version",
			args: args{
				scope: &cloud.Scope{
					Cluster: &api.Cluster{
						ObjectMeta: v1.ObjectMeta{
							Name: "test",
						},
					},
					Logger: klogr.New(),
				},
			},
			wantErr: true,
			errmsg:  "missing Cluster version",
		}, {
			name: "unknown provider",
			args: args{
				scope: &cloud.Scope{
					Logger:        klogr.New(),
					StoreProvider: fake.New(),
					Cluster: &api.Cluster{
						ObjectMeta: v1.ObjectMeta{
							Name: "unknown",
						},
						Spec: api.PharmerClusterSpec{
							Config: api.ClusterConfig{
								Cloud: api.CloudSpec{
									CloudProvider: "unknown",
									Zone:          "us",
								},
								KubernetesVersion: "1.13.4",
							},
						},
					},
				},
			},
			wantErr: true,
			errmsg:  "cloud provider not registerd",
		}, {
			name: "gce Cluster",
			args: args{
				scope: &cloud.Scope{
					Logger:        klogr.New(),
					StoreProvider: fake.New(),
					Cluster: &api.Cluster{
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
			},
			wantErr:    false,
			beforeTest: genericBeforeTest,
		}, {
			name: "aws Cluster",
			args: args{
				scope: &cloud.Scope{
					Logger:        klogr.New(),
					StoreProvider: fake.New(),
					Cluster: &api.Cluster{
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
			},
			wantErr:    false,
			beforeTest: genericBeforeTest,
		}, {
			name: "azure Cluster",
			args: args{
				scope: &cloud.Scope{
					Logger:        klogr.New(),
					StoreProvider: fake.New(),
					Cluster: &api.Cluster{
						ObjectMeta: v1.ObjectMeta{
							Name: "azure",
						},
						Spec: api.PharmerClusterSpec{
							ClusterAPI: v1alpha1.Cluster{},
							Config: api.ClusterConfig{
								MasterCount: 3,
								Cloud: api.CloudSpec{
									CloudProvider: "azure",
									Zone:          "us-east",
								},
								CredentialName:    "azure-cred",
								KubernetesVersion: "1.13.1",
							},
						},
					},
				},
			},
			wantErr: false,
			beforeTest: func(t *testing.T, a args) func(t *testing.T) {
				s := a.scope
				g := gomega.NewGomegaWithT(t)
				_, err := s.StoreProvider.Credentials().Create(&cloudapi.Credential{
					ObjectMeta: v1.ObjectMeta{
						Name: "azure-cred",
					},
					Spec: cloudapi.CredentialSpec{
						Provider: "azure",
						Data: map[string]string{
							"clientID":       "a",
							"clientSecret":   "b",
							"subscriptionID": "c",
							"tenantID":       "d",
						},
					},
				})
				g.Expect(err).NotTo(gomega.HaveOccurred())

				return func(t *testing.T) {
					s := a.scope
					// check if load balancer ip is set
					s.Cluster.Spec.Config.APIServerCertSANs[0] = azure.DefaultInternalLBIPAddress

					// check if Cluster is created
					err := s.StoreProvider.Clusters().Delete(s.Cluster.Name)
					g.Expect(err).NotTo(gomega.HaveOccurred())

					// check certs are generated
					deleteCerts(t, s.StoreProvider, s.Cluster.Name)

					// check master machines are genereated
					for i := 0; i < s.Cluster.Spec.Config.MasterCount; i++ {
						err = s.StoreProvider.Machine(s.Cluster.Name).Delete(s.Cluster.MasterMachineName(i))
						g.Expect(err).NotTo(gomega.HaveOccurred())
					}
				}
			},
		}, {
			name: "gke Cluster",
			args: args{
				scope: &cloud.Scope{
					Logger:        klogr.New(),
					StoreProvider: fake.New(),
					Cluster: &api.Cluster{
						ObjectMeta: v1.ObjectMeta{
							Name: "gke",
						},
						Spec: api.PharmerClusterSpec{
							ClusterAPI: v1alpha1.Cluster{},
							Config: api.ClusterConfig{
								MasterCount: 3,
								Cloud: api.CloudSpec{
									CloudProvider: "gke",
									Zone:          "us-central-1f",
								},
								KubernetesVersion: "1.13.1",
							},
						},
					},
				},
			},
			wantErr: false,
			beforeTest: func(t *testing.T, a args) func(t *testing.T) {
				g := gomega.NewGomegaWithT(t)
				return func(t *testing.T) {
					s := a.scope
					// check if Cluster is created
					err := s.StoreProvider.Clusters().Delete(s.Cluster.Name)
					g.Expect(err).NotTo(gomega.HaveOccurred())

					// check certs are generated
					deleteCerts(t, s.StoreProvider, s.Cluster.Name)
				}
			},
		}, {
			name: "aks Cluster",
			args: args{
				scope: &cloud.Scope{
					Logger:        klogr.New(),
					StoreProvider: fake.New(),
					Cluster: &api.Cluster{
						ObjectMeta: v1.ObjectMeta{
							Name: "aks",
						},
						Spec: api.PharmerClusterSpec{
							ClusterAPI: v1alpha1.Cluster{},
							Config: api.ClusterConfig{
								MasterCount: 3,
								Cloud: api.CloudSpec{
									CloudProvider: "aks",
									Zone:          "useast2",
								},
								KubernetesVersion: "1.13.1",
							},
						},
					},
				},
			},
			wantErr:    false,
			beforeTest: genericBeforeTest,
		}, {
			name: "linode Cluster",
			args: args{
				scope: &cloud.Scope{
					Logger:        klogr.New(),
					StoreProvider: fake.New(),
					Cluster: &api.Cluster{
						ObjectMeta: v1.ObjectMeta{
							Name: "linode",
						},
						Spec: api.PharmerClusterSpec{
							ClusterAPI: v1alpha1.Cluster{},
							Config: api.ClusterConfig{
								MasterCount: 3,
								Cloud: api.CloudSpec{
									CloudProvider: "linode",
									Zone:          "us-east",
								},
								KubernetesVersion: "1.13.1",
							},
						},
					},
				},
			},
			wantErr:    false,
			beforeTest: genericBeforeTest,
		}, {
			name: "digitalocean Cluster",
			args: args{
				scope: &cloud.Scope{
					Logger:        klogr.New(),
					StoreProvider: fake.New(),
					Cluster: &api.Cluster{
						ObjectMeta: v1.ObjectMeta{
							Name: "digitalocean",
						},
						Spec: api.PharmerClusterSpec{
							ClusterAPI: v1alpha1.Cluster{},
							Config: api.ClusterConfig{
								MasterCount: 3,
								Cloud: api.CloudSpec{
									CloudProvider: "digitalocean",
									Zone:          "nyc1",
								},
								KubernetesVersion: "1.13.1",
							},
						},
					},
				},
			},
			wantErr:    false,
			beforeTest: genericBeforeTest,
		}, {
			name: "packet Cluster",
			args: args{
				scope: &cloud.Scope{
					Logger:        klogr.New(),
					StoreProvider: fake.New(),
					Cluster: &api.Cluster{
						ObjectMeta: v1.ObjectMeta{
							Name: "packet",
						},
						Spec: api.PharmerClusterSpec{
							ClusterAPI: v1alpha1.Cluster{},
							Config: api.ClusterConfig{
								MasterCount: 3,
								Cloud: api.CloudSpec{
									CloudProvider: "packet",
									Zone:          "ewr1",
								},
								KubernetesVersion: "1.13.1",
							},
						},
					},
				},
			},
			wantErr:    false,
			beforeTest: genericBeforeTest,
		}, {
			name: "dokube Cluster",
			args: args{
				scope: &cloud.Scope{
					Logger:        klogr.New(),
					StoreProvider: fake.New(),
					Cluster: &api.Cluster{
						ObjectMeta: v1.ObjectMeta{
							Name: "dokube",
						},
						Spec: api.PharmerClusterSpec{
							ClusterAPI: v1alpha1.Cluster{},
							Config: api.ClusterConfig{
								MasterCount: 3,
								Cloud: api.CloudSpec{
									CloudProvider: "dokube",
									Zone:          "nyc1",
								},
								KubernetesVersion: "1.13.1",
							},
						},
					},
				},
			},
			wantErr:    false,
			beforeTest: genericBeforeTest,
		}, {
			name: "eks Cluster",
			args: args{
				scope: &cloud.Scope{
					Logger:        klogr.New(),
					StoreProvider: fake.New(),
					Cluster: &api.Cluster{
						ObjectMeta: v1.ObjectMeta{
							Name: "eks",
						},
						Spec: api.PharmerClusterSpec{
							ClusterAPI: v1alpha1.Cluster{},
							Config: api.ClusterConfig{
								MasterCount: 3,
								Cloud: api.CloudSpec{
									CloudProvider: "eks",
									Zone:          "us-east-1b",
								},
								KubernetesVersion: "1.13.1",
							},
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

			err := cloud.CreateCluster(tt.args.scope)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateCluster() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				g.Expect(err.Error()).Should(gomega.ContainSubstring(tt.errmsg))
			}
		})
	}
}

func TestCreateMachineSets(t *testing.T) {
	type args struct {
		store store.ResourceInterface
		opts  *options.NodeGroupCreateConfig
	}
	genericBeforeTest := func(t *testing.T, a args, cluster *api.Cluster) func(*testing.T) {
		cluster, err := a.store.Clusters().Create(cluster)
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
	}

	tests := []struct {
		name       string
		args       args
		wantErr    bool
		cluster    *api.Cluster
		beforeTest func(t *testing.T, a args, cluster *api.Cluster) func(*testing.T)
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
			wantErr:    false,
			beforeTest: genericBeforeTest,
			cluster: &api.Cluster{
				ObjectMeta: v1.ObjectMeta{
					Name: "gce",
				},
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
			wantErr:    false,
			beforeTest: genericBeforeTest,
			cluster: &api.Cluster{
				ObjectMeta: v1.ObjectMeta{
					Name: "gce",
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
			wantErr:    false,
			beforeTest: genericBeforeTest,
			cluster: &api.Cluster{
				ObjectMeta: v1.ObjectMeta{
					Name: "aws",
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
			},
		}, {
			name: "test azure",
			args: args{
				store: fake.New(),
				opts: &options.NodeGroupCreateConfig{
					ClusterName: "azure",
					Nodes: map[string]int{
						"a": 1,
						"b": 2,
						"c": 3,
					},
				},
			},
			wantErr:    false,
			beforeTest: genericBeforeTest,
			cluster: &api.Cluster{
				ObjectMeta: v1.ObjectMeta{
					Name: "azure",
				},
				Spec: api.PharmerClusterSpec{
					Config: api.ClusterConfig{
						Cloud: api.CloudSpec{
							CloudProvider: "azure",
							Zone:          "useast2",
						},
						KubernetesVersion: "1.13.4",
					},
				},
			},
		}, {
			name: "test gke",
			args: args{
				store: fake.New(),
				opts: &options.NodeGroupCreateConfig{
					ClusterName: "gke",
					Nodes: map[string]int{
						"a": 1,
						"b": 2,
						"c": 3,
					},
				},
			},
			wantErr:    false,
			beforeTest: genericBeforeTest,
			cluster: &api.Cluster{
				ObjectMeta: v1.ObjectMeta{
					Name: "gke",
				},
				Spec: api.PharmerClusterSpec{
					Config: api.ClusterConfig{
						Cloud: api.CloudSpec{
							CloudProvider: "gke",
							Zone:          "us-central-1f",
						},
						KubernetesVersion: "1.13.4",
					},
				},
			},
		}, {

			name: "test aks",
			args: args{
				store: fake.New(),
				opts: &options.NodeGroupCreateConfig{
					ClusterName: "aks",
					Nodes: map[string]int{
						"a": 1,
						"b": 2,
						"c": 3,
					},
				},
			},
			wantErr:    false,
			beforeTest: genericBeforeTest,
			cluster: &api.Cluster{
				ObjectMeta: v1.ObjectMeta{
					Name: "aks",
				},
				Spec: api.PharmerClusterSpec{
					Config: api.ClusterConfig{
						Cloud: api.CloudSpec{
							CloudProvider: "aks",
							Zone:          "us-central-1f",
						},
						KubernetesVersion: "1.13.4",
					},
				},
			},
		}, {

			name: "test dokube",
			args: args{
				store: fake.New(),
				opts: &options.NodeGroupCreateConfig{
					ClusterName: "dokube",
					Nodes: map[string]int{
						"a": 1,
						"b": 2,
						"c": 3,
					},
				},
			},
			wantErr:    false,
			beforeTest: genericBeforeTest,
			cluster: &api.Cluster{
				ObjectMeta: v1.ObjectMeta{
					Name: "dokube",
				},
				Spec: api.PharmerClusterSpec{
					Config: api.ClusterConfig{
						Cloud: api.CloudSpec{
							CloudProvider: "dokube",
							Zone:          "nyc1",
						},
						KubernetesVersion: "1.13.4",
					},
				},
			},
		}, {

			name: "test eks",
			args: args{
				store: fake.New(),
				opts: &options.NodeGroupCreateConfig{
					ClusterName: "eks",
					Nodes: map[string]int{
						"a": 1,
						"b": 2,
						"c": 3,
					},
				},
			},
			wantErr:    false,
			beforeTest: genericBeforeTest,
			cluster: &api.Cluster{
				ObjectMeta: v1.ObjectMeta{
					Name: "eks",
				},
				Spec: api.PharmerClusterSpec{
					Config: api.ClusterConfig{
						Cloud: api.CloudSpec{
							CloudProvider: "eks",
							Zone:          "us-central-1f",
						},
						KubernetesVersion: "1.13.4",
					},
				},
			},
		}, {

			name: "test digitalocean",
			args: args{
				store: fake.New(),
				opts: &options.NodeGroupCreateConfig{
					ClusterName: "digitalocean",
					Nodes: map[string]int{
						"a": 1,
						"b": 2,
						"c": 3,
					},
				},
			},
			wantErr:    false,
			beforeTest: genericBeforeTest,
			cluster: &api.Cluster{
				ObjectMeta: v1.ObjectMeta{
					Name: "digitalocean",
				},
				Spec: api.PharmerClusterSpec{
					Config: api.ClusterConfig{
						Cloud: api.CloudSpec{
							CloudProvider: "digitalocean",
							Zone:          "nyc1",
						},
						KubernetesVersion: "1.13.4",
					},
				},
			},
		}, {

			name: "test linode",
			args: args{
				store: fake.New(),
				opts: &options.NodeGroupCreateConfig{
					ClusterName: "linode",
					Nodes: map[string]int{
						"a": 1,
						"b": 2,
						"c": 3,
					},
				},
			},
			wantErr:    false,
			beforeTest: genericBeforeTest,
			cluster: &api.Cluster{
				ObjectMeta: v1.ObjectMeta{
					Name: "linode",
				},
				Spec: api.PharmerClusterSpec{
					Config: api.ClusterConfig{
						Cloud: api.CloudSpec{
							CloudProvider: "linode",
							Zone:          "us-east",
						},
						KubernetesVersion: "1.13.4",
					},
				},
			},
		}, {

			name: "test packet",
			args: args{
				store: fake.New(),
				opts: &options.NodeGroupCreateConfig{
					ClusterName: "packet",
					Nodes: map[string]int{
						"a": 1,
						"b": 2,
						"c": 3,
					},
				},
			},
			wantErr:    false,
			beforeTest: genericBeforeTest,
			cluster: &api.Cluster{
				ObjectMeta: v1.ObjectMeta{
					Name: "packet",
				},
				Spec: api.PharmerClusterSpec{
					Config: api.ClusterConfig{
						Cloud: api.CloudSpec{
							CloudProvider: "packet",
							Zone:          "ewr1",
						},
						KubernetesVersion: "1.13.4",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			afterTest := tt.beforeTest(t, tt.args, tt.cluster)
			defer afterTest(t)

			if err := cloud.CreateMachineSets(tt.args.store, tt.args.opts); (err != nil) != tt.wantErr {
				t.Errorf("CreateMachineSets() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
