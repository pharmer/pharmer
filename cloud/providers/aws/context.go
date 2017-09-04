package aws

import (
	"context"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/errors"
	"github.com/appscode/go/types"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/phid"
	_ec2 "github.com/aws/aws-sdk-go/service/ec2"
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
	UID = "aws"
)

func init() {
	cloud.RegisterCloudManager(UID, func(ctx context.Context) (cloud.ClusterManager, error) { return New(ctx), nil })
}

func New(ctx context.Context) cloud.ClusterManager {
	return &ClusterManager{ctx: ctx}
}

func (cm *ClusterManager) GetInstance(md *api.InstanceStatus) (*api.Instance, error) {
	conn, err := NewConnector(cm.ctx, cm.cluster)
	if err != nil {
		return nil, err
	}
	cm.conn = conn
	i, err := cm.newKubeInstance(md.ExternalID)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// TODO: Role not set
	return i, nil
}

func (cm *ClusterManager) MatchInstance(i *api.Instance, md *api.InstanceStatus) bool {
	return i.Status.ExternalID == md.ExternalID
}

func (cm *ClusterManager) initCluster(req *proto.ClusterCreateRequest) error {
	var err error
	cm.namer = namer{cluster: cm.cluster}

	//cluster.Spec.ctx.Name = req.Name
	//cluster.Spec.ctx.PHID = phid.NewKubeCluster()
	//cluster.Spec.ctx.Provider = req.Provider
	//cluster.Spec.ctx.Zone = req.Zone

	cm.cluster.Spec.Region = cm.cluster.Spec.Zone[0 : len(cm.cluster.Spec.Zone)-1]
	cm.cluster.Spec.DoNotDelete = req.DoNotDelete

	cm.cluster.SetNodeGroups(req.NodeGroups)

	// https://github.com/kubernetes/kubernetes/blob/master/cluster/aws/config-default.sh#L33
	if cm.cluster.Spec.MasterSKU == "" {
		cm.cluster.Spec.MasterSKU = "m3.medium"
		if cm.cluster.NodeCount() > 5 {
			cm.cluster.Spec.MasterSKU = "m3.large"
		}
		if cm.cluster.NodeCount() > 10 {
			cm.cluster.Spec.MasterSKU = "m3.xlarge"
		}
		if cm.cluster.NodeCount() > 100 {
			cm.cluster.Spec.MasterSKU = "m3.2xlarge"
		}
		if cm.cluster.NodeCount() > 250 {
			cm.cluster.Spec.MasterSKU = "c4.4xlarge"
		}
		if cm.cluster.NodeCount() > 500 {
			cm.cluster.Spec.MasterSKU = "c4.8xlarge"
		}
	}

	cm.cluster.Spec.KubernetesMasterName = cm.namer.MasterName()
	cm.cluster.Spec.SSHKey, err = api.NewSSHKeyPair()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.cluster.Spec.SSHKeyExternalID = cm.namer.GenSSHKeyExternalID()
	cm.cluster.Spec.SSHKeyPHID = phid.NewSSHKey()

	cm.cluster.Spec.MasterSGName = cm.namer.GenMasterSGName()
	cm.cluster.Spec.NodeSGName = cm.namer.GenNodeSGName()

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

func (cm *ClusterManager) waitForInstanceState(instanceId string, state string) error {
	for {
		r1, err := cm.conn.ec2.DescribeInstances(&_ec2.DescribeInstancesInput{
			InstanceIds: []*string{types.StringP(instanceId)},
		})
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		curState := *r1.Reservations[0].Instances[0].State.Name
		if curState == state {
			break
		}
		cloud.Logger(cm.ctx).Infof("Waiting for instance %v to be %v (currently %v)", instanceId, state, curState)
		cloud.Logger(cm.ctx).Infof("Sleeping for 5 seconds...")
		time.Sleep(5 * time.Second)
	}
	return nil
}
