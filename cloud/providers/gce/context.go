package gce

import (
	"context"
	"strings"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/data/files"
	"github.com/appscode/pharmer/phid"
	semver "github.com/hashicorp/go-version"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	maxInstancesPerMIG = 5 // Should be 500
	defaultNetwork     = "default"
)

type ClusterManager struct {
	ctx     context.Context
	cluster *api.Cluster
	conn    *cloudConnector
	namer   namer
}

var _ cloud.ClusterManager = &ClusterManager{}

const (
	UID = "gce"
)

func init() {
	cloud.RegisterCloudManager(UID, func(ctx context.Context) (cloud.ClusterManager, error) { return New(ctx), nil })
}

func New(ctx context.Context) cloud.ClusterManager {
	return &ClusterManager{ctx: ctx}
}

func (cm *ClusterManager) MatchInstance(i *api.Instance, md *api.InstanceStatus) bool {
	return i.Name == md.Name
}

func NewCluster(req *proto.ClusterCreateRequest) (*api.Cluster, error) {
	kv, err := semver.NewVersion(req.KubernetesVersion)
	if err != nil {
		return nil, err
	}
	defaultSpec, err := files.GetDefaultClusterSpec(req.Provider, kv)
	if err != nil {
		return nil, err
	}
	cluster := &api.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:              req.Name,
			UID:               phid.NewKubeCluster(),
			CreationTimestamp: metav1.Time{Time: time.Now()},
		},
		Spec: *defaultSpec,
	}
	api.AssignTypeKind(cluster)
	namer := namer{cluster: cluster}

	cluster.Spec.Provider = req.Provider
	cluster.Spec.Zone = req.Zone
	cluster.Spec.CredentialName = req.CredentialUid
	cluster.Spec.Region = cluster.Spec.Zone[0:strings.LastIndex(cluster.Spec.Zone, "-")]
	cluster.Spec.DoNotDelete = req.DoNotDelete
	for _, ng := range req.NodeGroups {
		if ng.Count < 0 {
			ng.Count = 0
		}
		if ng.Count > maxInstancesPerMIG {
			ng.Count = maxInstancesPerMIG
		}
	}
	cluster.SetNodeGroups(req.NodeGroups)

	// check for instance count
	cluster.Spec.MasterSKU = "n1-standard-1"
	if cluster.NodeCount() > 5 {
		cluster.Spec.MasterSKU = "n1-standard-2"
	}
	if cluster.NodeCount() > 10 {
		cluster.Spec.MasterSKU = "n1-standard-4"
	}
	if cluster.NodeCount() > 100 {
		cluster.Spec.MasterSKU = "n1-standard-8"
	}
	if cluster.NodeCount() > 250 {
		cluster.Spec.MasterSKU = "n1-standard-16"
	}
	if cluster.NodeCount() > 500 {
		cluster.Spec.MasterSKU = "n1-standard-32"
	}

	// REGISTER_MASTER_KUBELET = false // always false, keep master lightweight
	// PREEMPTIBLE_NODE = false // Removed Support

	cluster.Spec.KubernetesMasterName = namer.MasterName()
	cluster.Spec.SSHKeyExternalID = namer.GenSSHKeyExternalID()

	cluster.Spec.KubeadmToken = cloud.GetKubeadmToken()
	cluster.Spec.KubernetesVersion = "v" + req.KubernetesVersion

	cluster.Spec.StartupConfigToken = rand.Characters(128)

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

	cluster.Spec.NonMasqueradeCIDR = "10.0.0.0/8"

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

func (cm *ClusterManager) updateContext() error {
	cm.cluster.Spec.GCECloudConfig = &api.GCECloudConfig{
		// TokenURL           :
		// TokenBody          :
		ProjectID:          cm.cluster.Spec.Project,
		NetworkName:        "default",
		NodeTags:           []string{cm.namer.NodePrefix()},
		NodeInstancePrefix: cm.namer.NodePrefix(),
		Multizone:          bool(cm.cluster.Spec.Multizone),
	}
	cm.cluster.Spec.CloudConfigPath = "/etc/gce.conf"
	cm.cluster.Spec.ClusterExternalDomain = cloud.Extra(cm.ctx).ExternalDomain(cm.cluster.Name)
	cm.cluster.Spec.ClusterInternalDomain = cloud.Extra(cm.ctx).InternalDomain(cm.cluster.Name)
	//if cm.ctx.AppsCodeClusterCreator == "" {
	//	cm.ctx.AppsCodeClusterCreator = cm.ctx.Auth.User.UserName
	//}
	cm.cluster.Spec.EnableWebhookTokenAuthentication = true
	cm.cluster.Spec.EnableAPIserverBasicAudit = true
	return nil
}
