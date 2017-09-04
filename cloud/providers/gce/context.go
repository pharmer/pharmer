package gce

import (
	"context"
	"strings"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/credential"
	"github.com/appscode/pharmer/phid"
	semver "github.com/hashicorp/go-version"
	oneliners "github.com/tamalsaha/go-oneliners"
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

func (cm *ClusterManager) initContext(req *proto.ClusterCreateRequest) error {
	oneliners.FILE()
	cm.namer = namer{cluster: cm.cluster}

	cm.cluster.Spec.Region = cm.cluster.Spec.Zone[0:strings.LastIndex(cm.cluster.Spec.Zone, "-")]
	cm.cluster.Spec.DoNotDelete = req.DoNotDelete

	for _, ng := range req.NodeGroups {
		if ng.Count < 0 {
			ng.Count = 0
		}
		if ng.Count > maxInstancesPerMIG {
			ng.Count = maxInstancesPerMIG
		}
	}
	cm.cluster.SetNodeGroups(req.NodeGroups)

	// TODO: Load once
	cred, err := cloud.Store(cm.ctx).Credentials().Get(cm.cluster.Spec.CredentialName)
	if err != nil {
		return err
	}
	typed := credential.GCE{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return errors.New().WithMessagef("Credential %s is invalid. Reason: %v", cm.cluster.Spec.CredentialName, err)
	}
	cm.cluster.Spec.Project = typed.ProjectID()

	// check for instance count
	cm.cluster.Spec.MasterSKU = "n1-standard-1"
	if cm.cluster.NodeCount() > 5 {
		cm.cluster.Spec.MasterSKU = "n1-standard-2"
	}
	if cm.cluster.NodeCount() > 10 {
		cm.cluster.Spec.MasterSKU = "n1-standard-4"
	}
	if cm.cluster.NodeCount() > 100 {
		cm.cluster.Spec.MasterSKU = "n1-standard-8"
	}
	if cm.cluster.NodeCount() > 250 {
		cm.cluster.Spec.MasterSKU = "n1-standard-16"
	}
	if cm.cluster.NodeCount() > 500 {
		cm.cluster.Spec.MasterSKU = "n1-standard-32"
	}

	// REGISTER_MASTER_KUBELET = false // always false, keep master lightweight
	// PREEMPTIBLE_NODE = false // Removed Support

	cm.cluster.Spec.KubernetesMasterName = cm.namer.MasterName()
	cm.cluster.Spec.SSHKey, err = api.NewSSHKeyPair()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.cluster.Spec.SSHKeyExternalID = cm.namer.GenSSHKeyExternalID()
	cm.cluster.Spec.SSHKeyPHID = phid.NewSSHKey()

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
	cm.cluster.Spec.KubeadmToken = cloud.GetKubeadmToken()
	cm.cluster.Spec.KubernetesVersion = "v" + req.KubernetesVersion

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
