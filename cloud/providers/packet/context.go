package packet

import (
	"context"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/credential"
	"github.com/appscode/pharmer/phid"
	semver "github.com/hashicorp/go-version"
)

type ClusterManager struct {
	ctx     context.Context
	cluster *api.Cluster
	conn    *cloudConnector
	namer   namer
}

var _ cloud.ClusterManager = &ClusterManager{}

const (
	UID = "packet"
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

func (cm *ClusterManager) initContext(req *proto.ClusterCreateRequest) error {
	var err error
	cm.namer = namer{cluster: cm.cluster}

	//cluster.Spec.ctx.Name = req.Name
	//cluster.Spec.ctx.PHID = phid.NewKubeCluster()
	//cluster.Spec.ctx.Provider = req.Provider
	//cluster.Spec.ctx.Zone = req.Zone

	cm.cluster.Spec.Region = cm.cluster.Spec.Zone
	cm.cluster.Spec.DoNotDelete = req.DoNotDelete

	cm.cluster.SetNodeGroups(req.NodeGroups)

	// TODO: Load once
	cred, err := cloud.Store(cm.ctx).Credentials().Get(cm.cluster.Spec.CredentialName)
	if err != nil {
		return err
	}
	typed := credential.Packet{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return errors.New().WithMessagef("Credential %s is invalid. Reason: %v", cm.cluster.Spec.CredentialName, err)
	}
	cm.cluster.Spec.Project = typed.ProjectID()

	cm.cluster.Spec.KubernetesMasterName = cm.namer.MasterName()
	cm.cluster.Spec.SSHKey, err = api.NewSSHKeyPair()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.cluster.Spec.SSHKeyExternalID = cm.namer.GenSSHKeyExternalID()
	cm.cluster.Spec.SSHKeyPHID = phid.NewSSHKey()

	cm.cluster.Spec.StartupConfigToken = rand.Characters(128)

	// TODO: FixIt!
	//cm.cluster.Spec.AppsCodeApiGrpcEndpoint = system.PublicAPIGrpcEndpoint()
	//cm.cluster.Spec.AppsCodeApiHttpEndpoint = system.PublicAPIHttpEndpoint()
	//cm.cluster.Spec.AppsCodeClusterRootDomain = system.ClusterBaseDomain()

	if cm.cluster.Spec.EnableWebhookTokenAuthentication {
		cm.cluster.Spec.AppscodeAuthnUrl = "" // TODO: FixIt system.KuberntesWebhookAuthenticationURL()
	}
	if cm.cluster.Spec.EnableWebhookTokenAuthorization {
		cm.cluster.Spec.AppscodeAuthzUrl = "" // TODO: FixIt system.KuberntesWebhookAuthorizationURL()
	}

	// TODO: FixIT!
	//cm.cluster.Spec.ClusterExternalDomain = Extra(ctx).ExternalDomain(cluster.Name)
	//cm.cluster.Spec.ClusterInternalDomain = Extra(ctx).InternalDomain(cluster.Name)
	//cluster.Status.Phase = api.ClusterPhasePending

	//-------------------------- ctx.MasterSKU = "94" // 2 cpu

	// Using custom image with memory controller enabled
	// -------------------------ctx.InstanceImage = "16604964" // "container-os-20160402" // Debian 8.4 x64

	cm.cluster.Spec.NonMasqueradeCIDR = "10.0.0.0/8"

	version, err := semver.NewVersion(cm.cluster.Spec.KubernetesVersion)
	if err != nil {
		version, err = semver.NewVersion(cm.cluster.Spec.KubernetesVersion)
		if err != nil {
			return err
		}
	}
	version = version.ToMutator().ResetPrerelease().ResetMetadata().Done()

	v_1_4, _ := semver.NewConstraint(">= 1.4")
	if v_1_4.Check(version) {
		// Enable ScheduledJobs: http://kubernetes.io/docs/user-guide/scheduled-jobs/#prerequisites
		/*if cm.cluster.Spec.EnableScheduledJobResource {
			if cm.cluster.Spec.RuntimeConfig == "" {
				cm.cluster.Spec.RuntimeConfig = "batch/v2alpha1"
			} else {
				cm.cluster.Spec.RuntimeConfig += ",batch/v2alpha1"
			}
		}*/

		// http://kubernetes.io/docs/admin/authentication/
		if cm.cluster.Spec.EnableWebhookTokenAuthentication {
			if cm.cluster.Spec.RuntimeConfig == "" {
				cm.cluster.Spec.RuntimeConfig = "authentication.k8s.io/v1beta1=true"
			} else {
				cm.cluster.Spec.RuntimeConfig += ",authentication.k8s.io/v1beta1=true"
			}
		}

		// http://kubernetes.io/docs/admin/authorization/
		if cm.cluster.Spec.EnableWebhookTokenAuthorization {
			if cm.cluster.Spec.RuntimeConfig == "" {
				cm.cluster.Spec.RuntimeConfig = "authorization.k8s.io/v1beta1=true"
			} else {
				cm.cluster.Spec.RuntimeConfig += ",authorization.k8s.io/v1beta1=true"
			}
		}
		if cm.cluster.Spec.EnableRBACAuthorization {
			if cm.cluster.Spec.RuntimeConfig == "" {
				cm.cluster.Spec.RuntimeConfig = "rbac.authorization.k8s.io/v1alpha1=true"
			} else {
				cm.cluster.Spec.RuntimeConfig += ",rbac.authorization.k8s.io/v1alpha1=true"
			}
		}
	}
	return nil
}
