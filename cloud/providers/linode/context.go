package linode

import (
	"context"
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

type ClusterManager struct {
	ctx     context.Context
	cluster *api.Cluster
	conn    *cloudConnector
	namer   namer
}

var _ cloud.ClusterManager = &ClusterManager{}

const (
	UID = "linode"
)

func init() {
	cloud.RegisterCloudManager(UID, func(ctx context.Context) (cloud.ClusterManager, error) { return New(ctx), nil })
}

func New(ctx context.Context) cloud.ClusterManager {
	return &ClusterManager{ctx: ctx}
}

func (cm *ClusterManager) Scale(req *proto.ClusterReconfigureRequest) error {
	return cloud.UnsupportedOperation
}

func (cm *ClusterManager) SetVersion(req *proto.ClusterReconfigureRequest) error {
	return cloud.UnsupportedOperation
}

func (cm *ClusterManager) GetInstance(md *api.InstanceStatus) (*api.Instance, error) {
	conn, err := NewConnector(cm.ctx, nil)
	if err != nil {
		return nil, err
	}
	im := &instanceManager{conn: conn}
	return im.GetInstance(md)
}

func (cm *ClusterManager) MatchInstance(i *api.Instance, md *api.InstanceStatus) bool {
	return i.Status.PrivateIP == md.PrivateIP
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
		ObjectMeta: api.ObjectMeta{
			Name:              req.Name,
			UID:               phid.NewKubeCluster(),
			CreationTimestamp: metav1.Time{Time: time.Now()},
		},
		Spec: *defaultSpec,
	}
	api.AssignTypeKind(cluster)
	namer := namer{cluster: cluster}

	cluster.Spec.Zone = req.Zone
	cluster.Spec.CredentialName = req.CredentialUid

	//cluster.Spec.ctx.Name = req.Name
	//cluster.Spec.ctx.PHID = phid.NewKubeCluster()
	//cluster.Spec.ctx.Provider = req.Provider
	//cluster.Spec.ctx.Zone = req.Zone

	cluster.Spec.Region = cluster.Spec.Zone
	cluster.Spec.DoNotDelete = req.DoNotDelete

	cluster.SetNodeGroups(req.NodeGroups)

	cluster.Spec.KubernetesMasterName = namer.MasterName()
	cluster.Spec.SSHKey, err = api.NewSSHKeyPair()
	if err != nil {
		return nil, err
	}
	cluster.Spec.SSHKeyExternalID = namer.GenSSHKeyExternalID()
	cluster.Spec.SSHKeyPHID = phid.NewSSHKey()

	cluster.Spec.StartupConfigToken = rand.Characters(128)

	// TODO: FixIt!
	//cluster.Spec.AppsCodeApiGrpcEndpoint = system.PublicAPIGrpcEndpoint()
	//cluster.Spec.AppsCodeApiHttpEndpoint = system.PublicAPIHttpEndpoint()
	//cluster.Spec.AppsCodeClusterRootDomain = system.ClusterBaseDomain()

	if cluster.Spec.EnableWebhookTokenAuthentication {
		cluster.Spec.AppscodeAuthnUrl = "" // TODO: FixIt system.KuberntesWebhookAuthenticationURL()
	}
	if cluster.Spec.EnableWebhookTokenAuthorization {
		cluster.Spec.AppscodeAuthzUrl = "" // TODO: FixIt system.KuberntesWebhookAuthorizationURL()
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
