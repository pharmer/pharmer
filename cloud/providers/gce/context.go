package gce

import (
	"encoding/base64"
	"strconv"
	"strings"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/context"
	"github.com/appscode/pharmer/credential"
	"github.com/appscode/pharmer/phid"
	semver "github.com/hashicorp/go-version"
	bstore "google.golang.org/api/storage/v1"
)

const (
	maxInstancesPerMIG = 5 // Should be 500
	defaultNetwork     = "default"
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

	cm.cluster.Region = cm.cluster.Zone[0:strings.LastIndex(cm.cluster.Zone, "-")]
	cm.cluster.DoNotDelete = req.DoNotDelete
	cm.cluster.BucketName = "kubernetes-" + cm.cluster.Name + "-" + rand.Characters(8)

	for _, ng := range req.NodeGroups {
		if ng.Count < 0 {
			ng.Count = 0
		}
		if ng.Count > maxInstancesPerMIG {
			ng.Count = maxInstancesPerMIG
		}
	}
	cm.cluster.SetNodeGroups(req.NodeGroups)
	cm.cluster.Project = req.GceProject
	if cm.cluster.Project == "" {
		cm.cluster.Project = cm.cluster.CloudCredential[credential.GCEProjectID]
	}

	// check for instance count
	cm.cluster.MasterSKU = "n1-standard-1"
	if cm.cluster.NodeCount() > 5 {
		cm.cluster.MasterSKU = "n1-standard-2"
	}
	if cm.cluster.NodeCount() > 10 {
		cm.cluster.MasterSKU = "n1-standard-4"
	}
	if cm.cluster.NodeCount() > 100 {
		cm.cluster.MasterSKU = "n1-standard-8"
	}
	if cm.cluster.NodeCount() > 250 {
		cm.cluster.MasterSKU = "n1-standard-16"
	}
	if cm.cluster.NodeCount() > 500 {
		cm.cluster.MasterSKU = "n1-standard-32"
	}

	// REGISTER_MASTER_KUBELET = false // always false, keep master lightweight
	// PREEMPTIBLE_NODE = false // Removed Support

	cm.cluster.KubernetesMasterName = cm.namer.MasterName()
	cm.cluster.SSHKey, err = api.NewSSHKeyPair()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.cluster.SSHKeyExternalID = cm.namer.GenSSHKeyExternalID()
	cm.cluster.SSHKeyPHID = phid.NewSSHKey()

	cloud.GenClusterTokens(cm.cluster)

	cm.cluster.GCECloudConfig = &api.GCECloudConfig{
		// TokenURL           :
		// TokenBody          :
		ProjectID:          cm.cluster.Project,
		NetworkName:        "default",
		NodeTags:           []string{cm.namer.NodePrefix()},
		NodeInstancePrefix: cm.namer.NodePrefix(),
		Multizone:          bool(cm.cluster.Multizone),
	}
	cm.cluster.CloudConfigPath = "/etc/gce.conf"
	cm.cluster.KubeadmToken = cloud.GetKubeadmToken()
	cm.cluster.KubernetesVersion = "v" + req.Version
	return nil
}

func (cm *clusterManager) updateContext() error {
	cm.cluster.GCECloudConfig = &api.GCECloudConfig{
		// TokenURL           :
		// TokenBody          :
		ProjectID:          cm.cluster.Project,
		NetworkName:        "default",
		NodeTags:           []string{cm.namer.NodePrefix()},
		NodeInstancePrefix: cm.namer.NodePrefix(),
		Multizone:          bool(cm.cluster.Multizone),
	}
	cm.cluster.CloudConfigPath = "/etc/gce.conf"
	cm.cluster.ClusterExternalDomain = cm.ctx.Extra().ExternalDomain(cm.cluster.Name)
	cm.cluster.ClusterInternalDomain = cm.ctx.Extra().InternalDomain(cm.cluster.Name)
	//if cm.ctx.AppsCodeClusterCreator == "" {
	//	cm.ctx.AppsCodeClusterCreator = cm.ctx.Auth.User.UserName
	//}
	cm.cluster.EnableWebhookTokenAuthentication = true
	cm.cluster.EnableApiserverBasicAudit = true
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
	cm.cluster.MasterSKU = "n1-standard-2"

	cm.cluster.MasterDiskType = "pd-standard" // "pd-ssd"
	cm.cluster.MasterDiskSize = 100
	cm.cluster.NodeDiskType = "pd-standard"
	cm.cluster.NodeDiskSize = 100

	// https://cloud.google.com/compute/docs/containers/container_vms
	// Comes pre installed with Docker and Kubelet
	cm.cluster.InstanceImage = "ubuntu-1604-xenial-v20170721" //"kube12-tamal"   // "debian-8-jessie-v20160219" // "container-vm-v20151215"
	cm.cluster.InstanceImageProject = "ubuntu-os-cloud"       //"k8s-dev" // "debian-cloud"              // "google-containers"

	// REGISTER_MASTER_KUBELET = false // always false, keep master lightweight

	// PREEMPTIBLE_NODE = false // Removed Support

	cm.cluster.MasterReservedIP = "auto" // GCE - change to "" for avoid allocating Elastic IP
	cm.cluster.MasterIPRange = "10.246.0.0/24"
	cm.cluster.ClusterIPRange = "10.244.0.0/16"
	cm.cluster.ServiceClusterIPRange = "10.0.0.0/16"
	cm.cluster.NodeScopes = []string{"compute-rw", "monitoring", "logging-write", "storage-ro"}
	cm.cluster.PollSleepInterval = 3

	cm.cluster.RegisterMasterKubelet = true
	cm.cluster.EnableNodePublicIP = true // from aws

	//gcs
	cm.cluster.AllocateNodeCIDRs = true

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

	// TODO(admin): Node autoscaler is always on, make it a choice
	cm.cluster.EnableNodeAutoscaler = false
	// cm.ctx.AutoscalerMinNodes = 1
	// cm.ctx.AutoscalerMaxNodes = 100
	cm.cluster.TargetNodeUtilization = 0.7

	cm.cluster.AdmissionControl = "NamespaceLifecycle,LimitRanger,ServiceAccount,ResourceQuota,PersistentVolumeLabel"
	// KUBE_UP_AUTOMATIC_CLEANUP

	cm.cluster.NetworkProvider = "none"
	cm.cluster.HairpinMode = "promiscuous-bridge"
	// cm.ctx.KubeletPort = "10250"

	version, err := semver.NewVersion(cm.cluster.KubernetesVersion)
	if err != nil {
		version, err = semver.NewVersion(cm.cluster.KubernetesVersion)
		if err != nil {
			return err
		}
	}
	version = version.ToBuilder().ResetPrerelease().ResetMetadata().Done()

	// https://github.com/appscode/kubernetes/blob/v1.3.6/cluster/gce/config-cloud.sh#L19
	v_1_3, _ := semver.NewConstraint(">= 1.3, < 1.4")
	if v_1_3.Check(version) {
		// Evict pods whenever compute resource availability on the nodes gets below a threshold.
		cm.cluster.EvictionHard = `memory.available<100Mi`

		cm.cluster.NetworkProvider = "kubenet"

		// Evict pods whenever compute resource availability on the nodes gets below a threshold.
		cm.cluster.EvictionHard = `memory.available<100Mi`
	}

	// https://github.com/appscode/kubernetes/blob/1.4.0-ac/cluster/gce/config-cloud.sh#L19
	v_1_4, _ := semver.NewConstraint(">= 1.4")
	if v_1_4.Check(version) {
		cm.cluster.ClusterIPRange = "10.244.0.0/14"
		cm.cluster.NetworkProvider = "kubenet"

		// Evict pods whenever compute resource availability on the nodes gets below a threshold.
		cm.cluster.EvictionHard = `memory.available<100Mi,nodefs.available<10%,nodefs.inodesFree<5%`

		cm.cluster.AdmissionControl = "NamespaceLifecycle,LimitRanger,ServiceAccount,PersistentVolumeLabel,DefaultStorageClass,ResourceQuota"
		cm.cluster.EnableRescheduler = true
	}

	cloud.BuildRuntimeConfig(cm.cluster)
	return nil
}

func (cm *clusterManager) UploadStartupConfig() error {
	if _, err := cm.conn.storageService.Buckets.Get(cm.cluster.BucketName).Do(); err != nil {
		_, err := cm.conn.storageService.Buckets.Insert(cm.cluster.Project, &bstore.Bucket{
			Name: cm.cluster.BucketName,
		}).Do()
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cm.ctx.Logger().Debug("Created bucket %s", cm.cluster.BucketName)
	} else {
		cm.ctx.Logger().Debug("Bucket %s already exists", cm.cluster.BucketName)
	}

	{
		caData := &bstore.Object{
			Name: "kubernetes/context/" + strconv.FormatInt(cm.cluster.ResourceVersion, 10) + "/pki/" + "ca.crt",
		}
		caCert, err := base64.StdEncoding.DecodeString(cm.cluster.CaCert)
		if _, err = cm.conn.storageService.Objects.Insert(cm.cluster.BucketName, caData).Media(strings.NewReader(string(caCert))).Do(); err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}

		caKeyData := &bstore.Object{
			Name: "kubernetes/context/" + strconv.FormatInt(cm.cluster.ResourceVersion, 10) + "/pki/" + "ca.key",
		}
		caKey, err := base64.StdEncoding.DecodeString(cm.cluster.CaKey)

		if _, err = cm.conn.storageService.Objects.Insert(cm.cluster.BucketName, caKeyData).Media(strings.NewReader(string(caKey))).Do(); err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		frontCAData := &bstore.Object{
			Name: "kubernetes/context/" + strconv.FormatInt(cm.cluster.ResourceVersion, 10) + "/pki/" + "front-proxy-ca.crt",
		}
		frontCACert, err := base64.StdEncoding.DecodeString(cm.cluster.FrontProxyCaCert)
		if _, err = cm.conn.storageService.Objects.Insert(cm.cluster.BucketName, frontCAData).Media(strings.NewReader(string(frontCACert))).Do(); err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		frontCAKeyData := &bstore.Object{
			Name: "kubernetes/context/" + strconv.FormatInt(cm.cluster.ResourceVersion, 10) + "/pki/" + "front-proxy-ca.key",
		}
		frontCAKey, err := base64.StdEncoding.DecodeString(cm.cluster.FrontProxyCaKey)
		if _, err = cm.conn.storageService.Objects.Insert(cm.cluster.BucketName, frontCAKeyData).Media(strings.NewReader(string(frontCAKey))).Do(); err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}

	}
	{
		cfg, err := cm.cluster.StartupConfigResponse(api.RoleKubernetesMaster)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		data := &bstore.Object{
			Name: "kubernetes/context/" + strconv.FormatInt(cm.cluster.ResourceVersion, 10) + "/startup-config/" + api.RoleKubernetesMaster + ".yaml",
		}
		_, err = cm.conn.storageService.Objects.Insert(cm.cluster.BucketName, data).Media(strings.NewReader(cfg)).Do()
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}
	{
		cfg, err := cm.cluster.StartupConfigResponse(api.RoleKubernetesPool)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		data := &bstore.Object{
			Name: "kubernetes/context/" + strconv.FormatInt(cm.cluster.ResourceVersion, 10) + "/startup-config/" + api.RoleKubernetesPool + ".yaml",
		}
		_, err = cm.conn.storageService.Objects.Insert(cm.cluster.BucketName, data).Media(strings.NewReader(cfg)).Do()
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}
	return nil
}
