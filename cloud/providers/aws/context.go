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
	"github.com/appscode/pharmer/util/kubeadm"
	_ec2 "github.com/aws/aws-sdk-go/service/ec2"
	_s3 "github.com/aws/aws-sdk-go/service/s3"
	semver "github.com/hashicorp/go-version"
)

type clusterManager struct {
	ctx     context.Context
	cluster *api.Cluster
	ins     *api.ClusterInstances
	conn    *cloudConnector
	namer   namer
}

func (cm *clusterManager) initContext(req *proto.ClusterCreateRequest) error {
	err := cm.LoadDefaultContext()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.namer = namer{cluster: cm.cluster}

	//cluster.ctx.Name = req.Name
	//cluster.ctx.PHID = phid.NewKubeCluster()
	//cluster.ctx.Provider = req.Provider
	//cluster.ctx.Zone = req.Zone

	cm.cluster.Region = cm.cluster.Zone[0 : len(cm.cluster.Zone)-1]
	cm.cluster.DoNotDelete = req.DoNotDelete
	cm.cluster.BucketName = "kubernetes-" + cm.cluster.Name + "-" + rand.Characters(8)

	cm.cluster.SetNodeGroups(req.NodeGroups)

	// https://github.com/kubernetes/kubernetes/blob/master/cluster/aws/config-default.sh#L33
	if cm.cluster.MasterSKU == "" {
		cm.cluster.MasterSKU = "m3.medium"
		if cm.cluster.NodeCount() > 5 {
			cm.cluster.MasterSKU = "m3.large"
		}
		if cm.cluster.NodeCount() > 10 {
			cm.cluster.MasterSKU = "m3.xlarge"
		}
		if cm.cluster.NodeCount() > 100 {
			cm.cluster.MasterSKU = "m3.2xlarge"
		}
		if cm.cluster.NodeCount() > 250 {
			cm.cluster.MasterSKU = "c4.4xlarge"
		}
		if cm.cluster.NodeCount() > 500 {
			cm.cluster.MasterSKU = "c4.8xlarge"
		}
	}

	cm.cluster.KubernetesMasterName = cm.namer.MasterName()
	cm.cluster.SSHKey, err = api.NewSSHKeyPair()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.cluster.SSHKeyExternalID = cm.namer.GenSSHKeyExternalID()
	cm.cluster.SSHKeyPHID = phid.NewSSHKey()

	cm.cluster.MasterSGName = cm.namer.GenMasterSGName()
	cm.cluster.NodeSGName = cm.namer.GenNodeSGName()

	cloud.GenClusterTokens(cm.cluster)

	cm.cluster.KubeadmToken = kubeadm.GetRandomToken()
	cm.cluster.KubeVersion = "v" + req.Version

	return nil
}

func (cm *clusterManager) LoadDefaultContext() error {
	err := cm.cluster.KubeEnv.SetDefaults()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	cm.cluster.ClusterExternalDomain = cm.ctx.Extra().ExternalDomain(cm.cluster.Name)
	cm.cluster.ClusterInternalDomain = cm.ctx.Extra().InternalDomain(cm.cluster.Name)

	cm.cluster.Status = api.KubernetesStatus_Pending
	cm.cluster.OS = "ubuntu"

	cm.cluster.AppsCodeLogIndexPrefix = "logstash-"
	cm.cluster.AppsCodeLogStorageLifetime = 90 * 24 * 3600
	cm.cluster.AppsCodeMonitoringStorageLifetime = 90 * 24 * 3600

	cm.cluster.DockerStorage = "aufs"

	cm.cluster.IAMProfileMaster = "kubernetes-master"
	cm.cluster.IAMProfileNode = "kubernetes-node"

	cm.cluster.MasterDiskType = "gp2"
	cm.cluster.MasterDiskSize = 100
	// cm.ctx.MasterDiskType = "gp2"
	// cm.ctx.MasterDiskSize = 8
	cm.cluster.NodeDiskType = "gp2"
	cm.cluster.NodeDiskSize = 100
	cm.cluster.NodeScopes = []string{}
	cm.cluster.PollSleepInterval = 3

	cm.cluster.ServiceClusterIPRange = "10.0.0.0/16"
	cm.cluster.ClusterIPRange = "10.244.0.0/16"
	cm.cluster.MasterIPRange = "10.246.0.0/24"
	cm.cluster.MasterReservedIP = "auto"

	cm.cluster.EnableClusterMonitoring = "appscode"
	cm.cluster.EnableNodeLogging = true
	cm.cluster.LoggingDestination = "appscode-elasticsearch"
	cm.cluster.EnableClusterLogging = true
	cm.cluster.ElasticsearchLoggingReplicas = 1

	cm.cluster.ExtraDockerOpts = ""

	cm.cluster.EnableClusterDNS = true
	cm.cluster.DNSServerIP = "10.0.0.10"
	cm.cluster.DNSDomain = "cluster.local"
	cm.cluster.DNSReplicas = 1

	// TODO: Needs multiple auto scaler
	cm.cluster.EnableNodeAutoscaler = false
	// cm.ctx.AutoscalerMinNodes = 1
	// cm.ctx.AutoscalerMaxNodes = 100
	cm.cluster.TargetNodeUtilization = 0.7

	cm.cluster.AdmissionControl = "NamespaceLifecycle,LimitRanger,ServiceAccount,ResourceQuota,PersistentVolumeLabel"
	// aws
	cm.cluster.RegisterMasterKubelet = true
	cm.cluster.EnableNodePublicIP = true

	cm.cluster.AllocateNodeCIDRs = true

	cm.cluster.VpcCidrBase = "172.20"
	cm.cluster.MasterIPSuffix = ".9"
	cm.cluster.MasterInternalIP = cm.cluster.VpcCidrBase + ".0" + cm.cluster.MasterIPSuffix

	cm.cluster.VpcCidr = cm.cluster.VpcCidrBase + ".0.0/16"
	cm.cluster.SubnetCidr = cm.cluster.VpcCidrBase + ".0.0/24"

	cm.cluster.NetworkProvider = "none"
	cm.cluster.HairpinMode = "promiscuous-bridge"
	cm.cluster.NonMasqueradeCidr = "10.0.0.0/8"

	version, err := semver.NewVersion(cm.cluster.KubeServerVersion)
	if err != nil {
		version, err = semver.NewVersion(cm.cluster.KubeVersion)
		if err != nil {
			return err
		}
	}
	version = version.ToBuilder().ResetPrerelease().ResetMetadata().Done()

	v_1_3, _ := semver.NewConstraint(">= 1.3, < 1.4")
	if v_1_3.Check(version) {
		cm.cluster.NetworkProvider = "kubenet"
	}

	v_1_4, _ := semver.NewConstraint(">= 1.4")
	if v_1_4.Check(version) {
		cm.cluster.NetworkProvider = "kubenet"
		cm.cluster.AdmissionControl = "NamespaceLifecycle,LimitRanger,ServiceAccount,PersistentVolumeLabel,DefaultStorageClass,ResourceQuota"
	}

	cloud.BuildRuntimeConfig(cm.cluster)
	return nil
}

func (cm *clusterManager) UploadStartupConfig() error {
	_, err := cm.conn.s3.GetBucketLocation(&_s3.GetBucketLocationInput{Bucket: types.StringP(cm.cluster.BucketName)})
	if err != nil {
		_, err = cm.conn.s3.CreateBucket(&_s3.CreateBucketInput{Bucket: types.StringP(cm.cluster.BucketName)})
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
		path := fmt.Sprintf("kubernetes/context/%v/startup-config/%v.yaml", cm.cluster.ContextVersion, api.RoleKubernetesMaster)
		params := &_s3.PutObjectInput{
			Bucket: types.StringP(cm.cluster.BucketName),
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
		caCert, err := base64.StdEncoding.DecodeString(cm.cluster.CaCert)
		path := fmt.Sprintf("kubernetes/context/%v/pki/ca.crt", cm.cluster.ContextVersion)
		if err = cm.bucketStore(path, caCert); err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		caKey, err := base64.StdEncoding.DecodeString(cm.cluster.CaKey)
		path = fmt.Sprintf("kubernetes/context/%v/pki/ca.key", cm.cluster.ContextVersion)
		if err = cm.bucketStore(path, caKey); err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		frontCACert, err := base64.StdEncoding.DecodeString(cm.cluster.FrontProxyCaCert)
		path = fmt.Sprintf("kubernetes/context/%v/pki/front-proxy-ca.crt", cm.cluster.ContextVersion)
		if err = cm.bucketStore(path, frontCACert); err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		frontCAKey, err := base64.StdEncoding.DecodeString(cm.cluster.FrontProxyCaKey)
		path = fmt.Sprintf("kubernetes/context/%v/pki/front-proxy-ca.key", cm.cluster.ContextVersion)
		if err = cm.bucketStore(path, frontCAKey); err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}
	{
		cfg, err := cm.cluster.StartupConfigResponse(api.RoleKubernetesPool)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		path := fmt.Sprintf("kubernetes/context/%v/startup-config/%v.yaml", cm.cluster.ContextVersion, api.RoleKubernetesPool)
		params := &_s3.PutObjectInput{
			Bucket: types.StringP(cm.cluster.BucketName),
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

func (cm *clusterManager) bucketStore(path string, data []byte) error {
	params := &_s3.PutObjectInput{
		Bucket: types.StringP(cm.cluster.BucketName),
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

func (cm *clusterManager) waitForInstanceState(instanceId string, state string) error {
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
