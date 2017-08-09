package gce

import (
	"strconv"
	"strings"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud/lib"
	"github.com/appscode/pharmer/contexts"
	"github.com/appscode/pharmer/extpoints"
	"github.com/appscode/pharmer/phid"
	"github.com/appscode/pharmer/storage"
	"github.com/appscode/pharmer/system"
	"github.com/appscode/pharmer/util/credentialutil"
	semver "github.com/hashicorp/go-version"
	bstore "google.golang.org/api/storage/v1"
)

func init() {
	extpoints.KubeProviders.Register(new(kubeProvider), "gce")
}

const (
	maxInstancesPerMIG = 5 // Should be 500
	defaultNetwork     = "default"
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

	cm.ctx.Region = cm.ctx.Zone[0:strings.LastIndex(cm.ctx.Zone, "-")]
	cm.ctx.DoNotDelete = req.DoNotDelete
	lib.SetApps(cm.ctx)
	cm.ctx.BucketName = "kubernetes-" + cm.ctx.Name + "-" + rand.Characters(8)

	for _, ng := range req.NodeGroups {
		if ng.Count < 0 {
			ng.Count = 0
		}
		if ng.Count > maxInstancesPerMIG {
			ng.Count = maxInstancesPerMIG
		}
	}
	cm.ctx.SetNodeGroups(req.NodeGroups)
	cm.ctx.Project = req.GceProject
	if cm.ctx.Project == "" {
		cm.ctx.Project = cm.ctx.CloudCredential[credentialutil.GCECredentialProjectID]
	}

	// check for instance count
	cm.ctx.MasterSKU = "n1-standard-1"
	if cm.ctx.NodeCount() > 5 {
		cm.ctx.MasterSKU = "n1-standard-2"
	}
	if cm.ctx.NodeCount() > 10 {
		cm.ctx.MasterSKU = "n1-standard-4"
	}
	if cm.ctx.NodeCount() > 100 {
		cm.ctx.MasterSKU = "n1-standard-8"
	}
	if cm.ctx.NodeCount() > 250 {
		cm.ctx.MasterSKU = "n1-standard-16"
	}
	if cm.ctx.NodeCount() > 500 {
		cm.ctx.MasterSKU = "n1-standard-32"
	}

	// REGISTER_MASTER_KUBELET = false // always false, keep master lightweight
	// PREEMPTIBLE_NODE = false // Removed Support

	cm.ctx.KubernetesMasterName = cm.namer.MasterName()
	cm.ctx.SSHKey, err = contexts.NewSSHKeyPair()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.SSHKeyExternalID = cm.namer.GenSSHKeyExternalID()
	cm.ctx.SSHKeyPHID = phid.NewSSHKey()

	lib.GenClusterTokens(cm.ctx)

	cm.ctx.AppsCodeNamespace = cm.ctx.Auth.Namespace

	cm.ctx.GCECloudConfig = &api.GCECloudConfig{
		// TokenURL           :
		// TokenBody          :
		ProjectID:          cm.ctx.Project,
		NetworkName:        "default",
		NodeTags:           []string{cm.namer.NodePrefix()},
		NodeInstancePrefix: cm.namer.NodePrefix(),
		Multizone:          bool(cm.ctx.Multizone),
	}
	cm.ctx.CloudConfigPath = "/etc/gce.conf"

	return nil
}

func (cm *clusterManager) updateContext() error {
	cm.ctx.GCECloudConfig = &api.GCECloudConfig{
		// TokenURL           :
		// TokenBody          :
		ProjectID:          cm.ctx.Project,
		NetworkName:        "default",
		NodeTags:           []string{cm.namer.NodePrefix()},
		NodeInstancePrefix: cm.namer.NodePrefix(),
		Multizone:          bool(cm.ctx.Multizone),
	}
	cm.ctx.CloudConfigPath = "/etc/gce.conf"
	cm.ctx.ClusterExternalDomain = system.ClusterExternalDomain(cm.ctx.Auth.Namespace, cm.ctx.Name)
	cm.ctx.ClusterInternalDomain = system.ClusterInternalDomain(cm.ctx.Auth.Namespace, cm.ctx.Name)
	if cm.ctx.AppsCodeClusterCreator == "" {
		cm.ctx.AppsCodeClusterCreator = cm.ctx.Auth.User.UserName
	}
	cm.ctx.EnableWebhookTokenAuthentication = true
	cm.ctx.EnableApiserverBasicAudit = true
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
	cm.ctx.MasterSKU = "n1-standard-2"

	cm.ctx.AppsCodeLogIndexPrefix = "logstash-"
	cm.ctx.AppsCodeLogStorageLifetime = 90 * 24 * 3600
	cm.ctx.AppsCodeMonitoringStorageLifetime = 90 * 24 * 3600

	cm.ctx.MasterDiskType = "pd-standard" // "pd-ssd"
	cm.ctx.MasterDiskSize = 100
	cm.ctx.NodeDiskType = "pd-standard"
	cm.ctx.NodeDiskSize = 100

	// https://cloud.google.com/compute/docs/containers/container_vms
	// Comes pre installed with Docker and Kubelet
	cm.ctx.InstanceImage = "kube12-tamal"   // "debian-8-jessie-v20160219" // "container-vm-v20151215"
	cm.ctx.InstanceImageProject = "k8s-dev" // "debian-cloud"              // "google-containers"

	// REGISTER_MASTER_KUBELET = false // always false, keep master lightweight

	// PREEMPTIBLE_NODE = false // Removed Support

	cm.ctx.MasterReservedIP = "auto" // GCE - change to "" for avoid allocating Elastic IP
	cm.ctx.MasterIPRange = "10.246.0.0/24"
	cm.ctx.ClusterIPRange = "10.244.0.0/16"
	cm.ctx.ServiceClusterIPRange = "10.0.0.0/16"
	cm.ctx.NodeScopes = []string{"compute-rw", "monitoring", "logging-write", "storage-ro"}
	cm.ctx.PollSleepInterval = 3

	cm.ctx.RegisterMasterKubelet = true
	cm.ctx.EnableNodePublicIP = true // from aws

	//gcs
	cm.ctx.AllocateNodeCIDRs = true

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

	// TODO(admin): Node autoscaler is always on, make it a choice
	cm.ctx.EnableNodeAutoscaler = false
	// cm.ctx.AutoscalerMinNodes = 1
	// cm.ctx.AutoscalerMaxNodes = 100
	cm.ctx.TargetNodeUtilization = 0.7

	cm.ctx.AdmissionControl = "NamespaceLifecycle,LimitRanger,ServiceAccount,ResourceQuota,PersistentVolumeLabel"
	// KUBE_UP_AUTOMATIC_CLEANUP

	cm.ctx.NetworkProvider = "none"
	cm.ctx.HairpinMode = "promiscuous-bridge"
	// cm.ctx.KubeletPort = "10250"

	version, err := semver.NewVersion(cm.ctx.KubeServerVersion)
	if err != nil {
		version, err = semver.NewVersion(cm.ctx.KubeVersion)
		if err != nil {
			return err
		}
	}
	version = version.ToBuilder().ResetPrerelease().ResetMetadata().Done()

	// https://github.com/appscode/kubernetes/blob/v1.3.6/cluster/gce/config-lib.sh#L19
	v_1_3, _ := semver.NewConstraint(">= 1.3, < 1.4")
	if v_1_3.Check(version) {
		// Evict pods whenever compute resource availability on the nodes gets below a threshold.
		cm.ctx.EvictionHard = `memory.available<100Mi`

		cm.ctx.NetworkProvider = "kubenet"

		// Evict pods whenever compute resource availability on the nodes gets below a threshold.
		cm.ctx.EvictionHard = `memory.available<100Mi`
	}

	// https://github.com/appscode/kubernetes/blob/1.4.0-ac/cluster/gce/config-lib.sh#L19
	v_1_4, _ := semver.NewConstraint(">= 1.4")
	if v_1_4.Check(version) {
		cm.ctx.ClusterIPRange = "10.244.0.0/14"
		cm.ctx.NetworkProvider = "kubenet"

		// Evict pods whenever compute resource availability on the nodes gets below a threshold.
		cm.ctx.EvictionHard = `memory.available<100Mi,nodefs.available<10%,nodefs.inodesFree<5%`

		cm.ctx.AdmissionControl = "NamespaceLifecycle,LimitRanger,ServiceAccount,PersistentVolumeLabel,DefaultStorageClass,ResourceQuota"
		cm.ctx.EnableRescheduler = true
	}

	lib.BuildRuntimeConfig(cm.ctx)
	return nil
}

func (cm *clusterManager) UploadStartupConfig() error {
	if _, err := cm.conn.storageService.Buckets.Get(cm.ctx.BucketName).Do(); err != nil {
		_, err := cm.conn.storageService.Buckets.Insert(cm.ctx.Project, &bstore.Bucket{
			Name: cm.ctx.BucketName,
		}).Do()
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cm.ctx.Logger().Debug("Created bucket %s", cm.ctx.BucketName)
	} else {
		cm.ctx.Logger().Debug("Bucket %s already exists", cm.ctx.BucketName)
	}

	{
		cfg, err := cm.ctx.StartupConfigResponse(system.RoleKubernetesMaster)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		data := &bstore.Object{
			Name: "kubernetes/context/" + strconv.FormatInt(cm.ctx.ContextVersion, 10) + "/startup-config/" + system.RoleKubernetesMaster + ".yaml",
		}
		_, err = cm.conn.storageService.Objects.Insert(cm.ctx.BucketName, data).Media(strings.NewReader(cfg)).Do()
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}
	{
		cfg, err := cm.ctx.StartupConfigResponse(system.RoleKubernetesPool)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		data := &bstore.Object{
			Name: "kubernetes/context/" + strconv.FormatInt(cm.ctx.ContextVersion, 10) + "/startup-config/" + system.RoleKubernetesPool + ".yaml",
		}
		_, err = cm.conn.storageService.Objects.Insert(cm.ctx.BucketName, data).Media(strings.NewReader(cfg)).Do()
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}
	return nil
}
