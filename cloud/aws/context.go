package aws

import (
	"bytes"
	"fmt"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/types"
	"github.com/appscode/pharmer/cloud/common"
	"github.com/appscode/pharmer/contexts"
	"github.com/appscode/pharmer/phid"
	"github.com/appscode/pharmer/storage"
	"github.com/appscode/pharmer/system"
	_ec2 "github.com/aws/aws-sdk-go/service/ec2"
	_s3 "github.com/aws/aws-sdk-go/service/s3"
	semver "github.com/hashicorp/go-version"
)

type clusterManager struct {
	ctx   *contexts.ClusterContext
	ins   *contexts.ClusterInstances
	conn  *cloudConnector
	namer namer
}

func (cm *clusterManager) initContext(req *proto.ClusterCreateRequest) error {
	err := cm.LoadDefaultContext()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.namer = namer{ctx: cm.ctx}

	//cluster.ctx.Name = req.Name
	//cluster.ctx.PHID = phid.NewKubeCluster()
	//cluster.ctx.Provider = req.Provider
	//cluster.ctx.Zone = req.Zone

	cm.ctx.Region = cm.ctx.Zone[0 : len(cm.ctx.Zone)-1]
	cm.ctx.DoNotDelete = req.DoNotDelete
	common.SetApps(cm.ctx)
	cm.ctx.BucketName = "kubernetes-" + cm.ctx.Name + "-" + rand.Characters(8)

	cm.ctx.SetNodeGroups(req.NodeGroups)

	// https://github.com/kubernetes/kubernetes/blob/master/cluster/aws/config-default.sh#L33
	if cm.ctx.MasterSKU == "" {
		cm.ctx.MasterSKU = "m3.medium"
		if cm.ctx.NodeCount() > 5 {
			cm.ctx.MasterSKU = "m3.large"
		}
		if cm.ctx.NodeCount() > 10 {
			cm.ctx.MasterSKU = "m3.xlarge"
		}
		if cm.ctx.NodeCount() > 100 {
			cm.ctx.MasterSKU = "m3.2xlarge"
		}
		if cm.ctx.NodeCount() > 250 {
			cm.ctx.MasterSKU = "c4.4xlarge"
		}
		if cm.ctx.NodeCount() > 500 {
			cm.ctx.MasterSKU = "c4.8xlarge"
		}
	}

	cm.ctx.KubernetesMasterName = cm.namer.MasterName()
	cm.ctx.SSHKey, err = contexts.NewSSHKeyPair()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.SSHKeyExternalID = cm.namer.GenSSHKeyExternalID()
	cm.ctx.SSHKeyPHID = phid.NewSSHKey()

	cm.ctx.MasterSGName = cm.namer.GenMasterSGName()
	cm.ctx.NodeSGName = cm.namer.GenNodeSGName()

	common.GenClusterTokens(cm.ctx)

	cm.ctx.AppsCodeNamespace = cm.ctx.Auth.Namespace

	return nil
}

func (cm *clusterManager) LoadDefaultContext() error {
	err := cm.ctx.KubeEnv.SetDefaults()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	cm.ctx.ClusterExternalDomain = system.ClusterExternalDomain(cm.ctx.Auth.Namespace, cm.ctx.Name)
	cm.ctx.ClusterInternalDomain = system.ClusterInternalDomain(cm.ctx.Auth.Namespace, cm.ctx.Name)

	cm.ctx.Status = storage.KubernetesStatus_Pending
	cm.ctx.OS = "debian"

	cm.ctx.AppsCodeLogIndexPrefix = "logstash-"
	cm.ctx.AppsCodeLogStorageLifetime = 90 * 24 * 3600
	cm.ctx.AppsCodeMonitoringStorageLifetime = 90 * 24 * 3600

	cm.ctx.DockerStorage = "aufs"

	cm.ctx.IAMProfileMaster = "kubernetes-master"
	cm.ctx.IAMProfileNode = "kubernetes-node"

	cm.ctx.MasterDiskType = "gp2"
	cm.ctx.MasterDiskSize = 100
	// cm.ctx.MasterDiskType = "gp2"
	// cm.ctx.MasterDiskSize = 8
	cm.ctx.NodeDiskType = "gp2"
	cm.ctx.NodeDiskSize = 100
	cm.ctx.NodeScopes = []string{}
	cm.ctx.PollSleepInterval = 3

	cm.ctx.ServiceClusterIPRange = "10.0.0.0/16"
	cm.ctx.ClusterIPRange = "10.244.0.0/16"
	cm.ctx.MasterIPRange = "10.246.0.0/24"
	cm.ctx.MasterReservedIP = "auto"

	cm.ctx.EnableClusterMonitoring = "appscode"
	cm.ctx.EnableNodeLogging = true
	cm.ctx.LoggingDestination = "appscode-elasticsearch"
	cm.ctx.EnableClusterLogging = true
	cm.ctx.ElasticsearchLoggingReplicas = 1

	cm.ctx.ExtraDockerOpts = ""

	cm.ctx.EnableClusterDNS = true
	cm.ctx.DNSServerIP = "10.0.0.10"
	cm.ctx.DNSDomain = "cluster.local"
	cm.ctx.DNSReplicas = 1

	// TODO: Needs multiple auto scaler
	cm.ctx.EnableNodeAutoscaler = false
	// cm.ctx.AutoscalerMinNodes = 1
	// cm.ctx.AutoscalerMaxNodes = 100
	cm.ctx.TargetNodeUtilization = 0.7

	cm.ctx.AdmissionControl = "NamespaceLifecycle,LimitRanger,ServiceAccount,ResourceQuota,PersistentVolumeLabel"
	// aws
	cm.ctx.RegisterMasterKubelet = true
	cm.ctx.EnableNodePublicIP = true

	cm.ctx.AllocateNodeCIDRs = true

	cm.ctx.VpcCidrBase = "172.20"
	cm.ctx.MasterIPSuffix = ".9"
	cm.ctx.MasterInternalIP = cm.ctx.VpcCidrBase + ".0" + cm.ctx.MasterIPSuffix

	cm.ctx.VpcCidr = cm.ctx.VpcCidrBase + ".0.0/16"
	cm.ctx.SubnetCidr = cm.ctx.VpcCidrBase + ".0.0/24"

	cm.ctx.NetworkProvider = "none"
	cm.ctx.HairpinMode = "promiscuous-bridge"
	cm.ctx.NonMasqueradeCidr = "10.0.0.0/8"

	version, err := semver.NewVersion(cm.ctx.KubeServerVersion)
	if err != nil {
		version, err = semver.NewVersion(cm.ctx.KubeVersion)
		if err != nil {
			return err
		}
	}
	version = version.ToBuilder().ResetPrerelease().ResetMetadata().Done()

	v_1_3, _ := semver.NewConstraint(">= 1.3, < 1.4")
	if v_1_3.Check(version) {
		cm.ctx.NetworkProvider = "kubenet"
	}

	v_1_4, _ := semver.NewConstraint(">= 1.4")
	if v_1_4.Check(version) {
		cm.ctx.NetworkProvider = "kubenet"
		cm.ctx.AdmissionControl = "NamespaceLifecycle,LimitRanger,ServiceAccount,PersistentVolumeLabel,DefaultStorageClass,ResourceQuota"
	}

	common.BuildRuntimeConfig(cm.ctx)
	return nil
}

func (cm *clusterManager) UploadStartupConfig() error {
	_, err := cm.conn.s3.GetBucketLocation(&_s3.GetBucketLocationInput{Bucket: types.StringP(cm.ctx.BucketName)})
	if err != nil {
		_, err = cm.conn.s3.CreateBucket(&_s3.CreateBucketInput{Bucket: types.StringP(cm.ctx.BucketName)})
		if err != nil {
			cm.ctx.Logger().Debugf("Bucket name is no unique")
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}

	{
		cfg, err := cm.ctx.StartupConfigResponse(system.RoleKubernetesMaster)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		path := fmt.Sprintf("kubernetes/context/%v/startup-config/%v.yaml", cm.ctx.ContextVersion, system.RoleKubernetesMaster)
		params := &_s3.PutObjectInput{
			Bucket: types.StringP(cm.ctx.BucketName),
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
		cfg, err := cm.ctx.StartupConfigResponse(system.RoleKubernetesPool)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		path := fmt.Sprintf("kubernetes/context/%v/startup-config/%v.yaml", cm.ctx.ContextVersion, system.RoleKubernetesPool)
		params := &_s3.PutObjectInput{
			Bucket: types.StringP(cm.ctx.BucketName),
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

func (cluster *clusterManager) waitForInstanceState(instanceId string, state string) error {
	for {
		r1, err := cluster.conn.ec2.DescribeInstances(&_ec2.DescribeInstancesInput{
			InstanceIds: []*string{types.StringP(instanceId)},
		})
		if err != nil {
			return errors.FromErr(err).WithContext(cluster.ctx).Err()
		}
		curState := *r1.Reservations[0].Instances[0].State.Name
		if curState == state {
			break
		}
		cluster.ctx.Logger().Infof("Waiting for instance %v to be %v (currently %v)", instanceId, state, curState)
		cluster.ctx.Logger().Infof("Sleeping for 5 seconds...")
		time.Sleep(5 * time.Second)
	}
	return nil
}
