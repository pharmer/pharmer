package aws

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/types"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/context"
	"github.com/appscode/pharmer/phid"
	_ec2 "github.com/aws/aws-sdk-go/service/ec2"
	_s3 "github.com/aws/aws-sdk-go/service/s3"
	semver "github.com/hashicorp/go-version"
)

type ClusterManager struct {
	ctx     context.Context
	cluster *api.Cluster
	ins     *api.ClusterInstances
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

func (cm *ClusterManager) GetInstance(md *api.InstanceMetadata) (*api.Instance, error) {
	conn, err := NewConnector(nil)
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

func (cm *ClusterManager) MatchInstance(i *api.Instance, md *api.InstanceMetadata) bool {
	return i.Status.ExternalID == md.ExternalID
}

func (cm *ClusterManager) initContext(req *proto.ClusterCreateRequest) error {
	err := cm.LoadDefaultContext()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.namer = namer{cluster: cm.cluster}

	//cluster.Spec.ctx.Name = req.Name
	//cluster.Spec.ctx.PHID = phid.NewKubeCluster()
	//cluster.Spec.ctx.Provider = req.Provider
	//cluster.Spec.ctx.Zone = req.Zone

	cm.cluster.Spec.Region = cm.cluster.Spec.Zone[0 : len(cm.cluster.Spec.Zone)-1]
	cm.cluster.Spec.DoNotDelete = req.DoNotDelete
	cm.cluster.Spec.BucketName = "kubernetes-" + cm.cluster.Name + "-" + rand.Characters(8)

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

	cloud.GenClusterTokens(cm.cluster)

	cm.cluster.Spec.KubeadmToken = cloud.GetKubeadmToken()
	cm.cluster.Spec.KubernetesVersion = "v" + req.KubernetesVersion

	return nil
}

func (cm *ClusterManager) LoadDefaultContext() error {
	err := cm.cluster.Spec.KubeEnv.SetDefaults()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	cm.cluster.Spec.ClusterExternalDomain = cm.ctx.Extra().ExternalDomain(cm.cluster.Name)
	cm.cluster.Spec.ClusterInternalDomain = cm.ctx.Extra().InternalDomain(cm.cluster.Name)

	cm.cluster.Status.Phase = api.ClusterPhasePending
	cm.cluster.Spec.OS = "ubuntu"

	cm.cluster.Spec.DockerStorage = "aufs"

	cm.cluster.Spec.IAMProfileMaster = "kubernetes-master"
	cm.cluster.Spec.IAMProfileNode = "kubernetes-node"

	cm.cluster.Spec.MasterDiskType = "gp2"
	cm.cluster.Spec.MasterDiskSize = 100
	// cm.ctx.MasterDiskType = "gp2"
	// cm.ctx.MasterDiskSize = 8
	cm.cluster.Spec.NodeDiskType = "gp2"
	cm.cluster.Spec.NodeDiskSize = 100
	cm.cluster.Spec.NodeScopes = []string{}
	cm.cluster.Spec.PollSleepInterval = 3

	cm.cluster.Spec.ServiceClusterIPRange = "10.0.0.0/16"
	cm.cluster.Spec.ClusterIPRange = "10.244.0.0/16"
	cm.cluster.Spec.MasterIPRange = "10.246.0.0/24"
	cm.cluster.Spec.MasterReservedIP = "auto"

	cm.cluster.Spec.EnableClusterMonitoring = "appscode"
	cm.cluster.Spec.EnableNodeLogging = true
	cm.cluster.Spec.LoggingDestination = "appscode-elasticsearch"
	cm.cluster.Spec.EnableClusterLogging = true
	cm.cluster.Spec.ElasticsearchLoggingReplicas = 1

	cm.cluster.Spec.ExtraDockerOpts = ""

	cm.cluster.Spec.EnableClusterDNS = true
	cm.cluster.Spec.DNSServerIP = "10.0.0.10"
	cm.cluster.Spec.DNSDomain = "cluster.Spec.local"
	cm.cluster.Spec.DNSReplicas = 1

	// TODO: Needs multiple auto scaler
	cm.cluster.Spec.EnableNodeAutoscaler = false
	// cm.ctx.AutoscalerMinNodes = 1
	// cm.ctx.AutoscalerMaxNodes = 100
	cm.cluster.Spec.TargetNodeUtilization = 0.7

	cm.cluster.Spec.AdmissionControl = "NamespaceLifecycle,LimitRanger,ServiceAccount,ResourceQuota,PersistentVolumeLabel"
	// aws
	cm.cluster.Spec.RegisterMasterKubelet = true
	cm.cluster.Spec.EnableNodePublicIP = true

	cm.cluster.Spec.AllocateNodeCIDRs = true

	cm.cluster.Spec.VpcCidrBase = "172.20"
	cm.cluster.Spec.MasterIPSuffix = ".9"
	cm.cluster.Spec.MasterInternalIP = cm.cluster.Spec.VpcCidrBase + ".0" + cm.cluster.Spec.MasterIPSuffix

	cm.cluster.Spec.VpcCidr = cm.cluster.Spec.VpcCidrBase + ".0.0/16"
	cm.cluster.Spec.SubnetCidr = cm.cluster.Spec.VpcCidrBase + ".0.0/24"

	cm.cluster.Spec.NetworkProvider = "none"
	cm.cluster.Spec.HairpinMode = "promiscuous-bridge"
	cm.cluster.Spec.NonMasqueradeCidr = "10.0.0.0/8"

	version, err := semver.NewVersion(cm.cluster.Spec.KubernetesVersion)
	if err != nil {
		version, err = semver.NewVersion(cm.cluster.Spec.KubernetesVersion)
		if err != nil {
			return err
		}
	}
	version = version.ToBuilder().ResetPrerelease().ResetMetadata().Done()

	v_1_3, _ := semver.NewConstraint(">= 1.3, < 1.4")
	if v_1_3.Check(version) {
		cm.cluster.Spec.NetworkProvider = "kubenet"
	}

	v_1_4, _ := semver.NewConstraint(">= 1.4")
	if v_1_4.Check(version) {
		cm.cluster.Spec.NetworkProvider = "kubenet"
		cm.cluster.Spec.AdmissionControl = "NamespaceLifecycle,LimitRanger,ServiceAccount,PersistentVolumeLabel,DefaultStorageClass,ResourceQuota"
	}

	cloud.BuildRuntimeConfig(cm.cluster)
	return nil
}

func (cm *ClusterManager) UploadStartupConfig() error {
	_, err := cm.conn.s3.GetBucketLocation(&_s3.GetBucketLocationInput{Bucket: types.StringP(cm.cluster.Spec.BucketName)})
	if err != nil {
		_, err = cm.conn.s3.CreateBucket(&_s3.CreateBucketInput{Bucket: types.StringP(cm.cluster.Spec.BucketName)})
		if err != nil {
			cm.ctx.Logger().Debugf("Bucket name is no unique")
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}

	{
		cfg, err := cm.cluster.StartupConfigResponse(api.RoleKubernetesMaster)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		path := fmt.Sprintf("kubernetes/context/%v/startup-config/%v.yaml", cm.cluster.Spec.ResourceVersion, api.RoleKubernetesMaster)
		params := &_s3.PutObjectInput{
			Bucket: types.StringP(cm.cluster.Spec.BucketName),
			Key:    types.StringP(path),
			ACL:    types.StringP("authenticated-read"),
			Body:   bytes.NewReader([]byte(cfg)),
		}
		_, err = cm.conn.s3.PutObject(params)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}
	{
		caCert, err := base64.StdEncoding.DecodeString(cm.cluster.Spec.CaCert)
		path := fmt.Sprintf("kubernetes/context/%v/pki/ca.crt", cm.cluster.Spec.ResourceVersion)
		if err = cm.bucketStore(path, caCert); err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		caKey, err := base64.StdEncoding.DecodeString(cm.cluster.Spec.CaKey)
		path = fmt.Sprintf("kubernetes/context/%v/pki/ca.key", cm.cluster.Spec.ResourceVersion)
		if err = cm.bucketStore(path, caKey); err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		frontCACert, err := base64.StdEncoding.DecodeString(cm.cluster.Spec.FrontProxyCaCert)
		path = fmt.Sprintf("kubernetes/context/%v/pki/front-proxy-ca.crt", cm.cluster.Spec.ResourceVersion)
		if err = cm.bucketStore(path, frontCACert); err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		frontCAKey, err := base64.StdEncoding.DecodeString(cm.cluster.Spec.FrontProxyCaKey)
		path = fmt.Sprintf("kubernetes/context/%v/pki/front-proxy-ca.key", cm.cluster.Spec.ResourceVersion)
		if err = cm.bucketStore(path, frontCAKey); err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}
	{
		cfg, err := cm.cluster.StartupConfigResponse(api.RoleKubernetesPool)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		path := fmt.Sprintf("kubernetes/context/%v/startup-config/%v.yaml", cm.cluster.Spec.ResourceVersion, api.RoleKubernetesPool)
		params := &_s3.PutObjectInput{
			Bucket: types.StringP(cm.cluster.Spec.BucketName),
			Key:    types.StringP(path),
			ACL:    types.StringP("authenticated-read"),
			Body:   bytes.NewReader([]byte(cfg)),
		}
		_, err = cm.conn.s3.PutObject(params)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}
	return nil
}

func (cm *ClusterManager) bucketStore(path string, data []byte) error {
	params := &_s3.PutObjectInput{
		Bucket: types.StringP(cm.cluster.Spec.BucketName),
		Key:    types.StringP(path),
		ACL:    types.StringP("authenticated-read"),
		Body:   bytes.NewReader(data),
	}
	_, err := cm.conn.s3.PutObject(params)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
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
		cm.ctx.Logger().Infof("Waiting for instance %v to be %v (currently %v)", instanceId, state, curState)
		cm.ctx.Logger().Infof("Sleeping for 5 seconds...")
		time.Sleep(5 * time.Second)
	}
	return nil
}
