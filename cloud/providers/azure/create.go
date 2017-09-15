package azure

import (
	"time"

	"github.com/appscode/mergo"
	"github.com/appscode/pharmer/api"
	. "github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/data/files"
	"github.com/appscode/pharmer/phid"
	semver "github.com/hashicorp/go-version"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (cm *ClusterManager) CreateMasterNodeGroup(cluster *api.Cluster) (*api.NodeGroup, error) {
	ig := api.NodeGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "master",
			ClusterName:       cluster.Name,
			UID:               phid.NewNodeGroup(),
			CreationTimestamp: metav1.Time{Time: time.Now()},
			Labels: map[string]string{
				"node-role.kubernetes.io/master": "true",
			},
		},
		Spec: api.NodeGroupSpec{
			Nodes: 1,
			Template: api.NodeTemplateSpec{
				Spec: api.NodeSpec{
					SKU:           "", // assign at the time of apply
					SpotInstances: false,
					DiskType:      "gp2",
					DiskSize:      128,
				},
			},
		},
	}
	return Store(cm.ctx).NodeGroups(cluster.Name).Create(&ig)
}

func (cm *ClusterManager) DefaultSpec(in *api.Cluster) (*api.Cluster, error) {
	// Load default spec from data files
	kv, err := semver.NewVersion(in.Spec.KubernetesVersion)
	if err != nil {
		return nil, err
	}
	defaultSpec, err := files.GetDefaultClusterSpec(in.Spec.Cloud.CloudProvider, kv)
	if err != nil {
		return nil, err
	}
	cluster := &api.Cluster{
		Spec: *defaultSpec,
	}

	// Copy default spec into return value
	err = mergo.MergeWithOverwrite(cluster, in)
	if err != nil {
		return nil, err
	}
	n := namer{cluster: cluster}

	// Init object meta
	cluster.ObjectMeta.UID = phid.NewKubeCluster()
	cluster.ObjectMeta.CreationTimestamp = metav1.Time{Time: time.Now()}
	cluster.ObjectMeta.Generation = time.Now().UnixNano()
	api.AssignTypeKind(cluster)

	// Init spec
	cluster.Spec.Cloud.Region = cluster.Spec.Cloud.Zone
	if cluster.Spec.Cloud.Azure == nil {
		cluster.Spec.Cloud.Azure = &api.AzureSpec{}
	}
	cluster.Spec.Token = GetKubeadmToken()
	cluster.Spec.Networking.NonMasqueradeCIDR = "10.0.0.0/8"
	cluster.Spec.KubernetesMasterName = n.MasterName()

	// Init status
	cluster.Status = api.ClusterStatus{
		Phase:            api.ClusterPending,
		SSHKeyExternalID: n.GenSSHKeyExternalID(),
	}

	// Fix the stuff below ----------------------------------------------------

	// cluster.Spec.ctx.MasterSGName = cluster.Spec.ctx.Name + "-master-" + rand.Characters(6)
	// cluster.Spec.ctx.NodeSGName = cluster.Spec.ctx.Name + "-node-" + rand.Characters(6)

	// TODO: FixIt!
	//cluster.Spec.AppsCodeApiGrpcEndpoint = system.PublicAPIGrpcEndpoint()
	//cluster.Spec.AppsCodeApiHttpEndpoint = system.PublicAPIHttpEndpoint()
	//cluster.Spec.AppsCodeClusterRootDomain = system.ClusterBaseDomain()

	if cluster.Spec.EnableWebhookTokenAuthentication {
		cluster.Spec.AppscodeAuthnURL = "" // TODO: FixIt system.KuberntesWebhookAuthenticationURL()
	}
	if cluster.Spec.EnableWebhookTokenAuthorization {
		cluster.Spec.AppscodeAuthzURL = "" // TODO: FixIt system.KuberntesWebhookAuthorizationURL()
	}

	// TODO: FixIT!
	//cluster.Spec.ClusterExternalDomain = Extra(ctx).ExternalDomain(cluster.Name)
	//cluster.Spec.ClusterInternalDomain = Extra(ctx).InternalDomain(cluster.Name)
	//cluster.Status.Phase = api.ClusterPhasePending

	//-------------------------- ctx.MasterSKU = "94" // 2 cpu

	// Using custom image with memory controller enabled
	// -------------------------ctx.InstanceImage = "16604964" // "container-os-20160402" // Debian 8.4 x64

	version, err := semver.NewVersion(cluster.Spec.KubernetesVersion)
	if err != nil {
		version, err = semver.NewVersion(cluster.Spec.KubernetesVersion)
		if err != nil {
			return nil, err
		}
	}
	version = version.ToMutator().ResetPrerelease().ResetMetadata().Done()

	v_1_4, _ := semver.NewConstraint(">= 1.4")
	if v_1_4.Check(version) {
		// Enable ScheduledJobs: http://kubernetes.io/docs/user-guide/scheduled-jobs/#prerequisites
		/*if cluster.Spec.EnableScheduledJobResource {
			if cluster.Spec.RuntimeConfig == "" {
				cluster.Spec.RuntimeConfig = "batch/v2alpha1"
			} else {
				cluster.Spec.RuntimeConfig += ",batch/v2alpha1"
			}
		}*/

		// http://kubernetes.io/docs/admin/authentication/
		if cluster.Spec.EnableWebhookTokenAuthentication {
			if cluster.Spec.RuntimeConfig == "" {
				cluster.Spec.RuntimeConfig = "authentication.k8s.io/v1beta1=true"
			} else {
				cluster.Spec.RuntimeConfig += ",authentication.k8s.io/v1beta1=true"
			}
		}

		// http://kubernetes.io/docs/admin/authorization/
		if cluster.Spec.EnableWebhookTokenAuthorization {
			if cluster.Spec.RuntimeConfig == "" {
				cluster.Spec.RuntimeConfig = "authorization.k8s.io/v1beta1=true"
			} else {
				cluster.Spec.RuntimeConfig += ",authorization.k8s.io/v1beta1=true"
			}
		}
		if cluster.Spec.EnableRBACAuthorization {
			if cluster.Spec.RuntimeConfig == "" {
				cluster.Spec.RuntimeConfig = "rbac.authorization.k8s.io/v1alpha1=true"
			} else {
				cluster.Spec.RuntimeConfig += ",rbac.authorization.k8s.io/v1alpha1=true"
			}
		}
	}
	return cluster, nil
}

func (cm *ClusterManager) IsValid(cluster *api.Cluster) (bool, error) {
	return false, UnsupportedOperation
}
