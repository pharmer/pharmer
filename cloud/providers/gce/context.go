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

	cm.cluster.Spec.Region = cm.cluster.Spec.Zone[0:strings.LastIndex(cm.cluster.Spec.Zone, "-")]
	cm.cluster.Spec.DoNotDelete = req.DoNotDelete
	cm.cluster.Spec.BucketName = "kubernetes-" + cm.cluster.Name + "-" + rand.Characters(8)

	for _, ng := range req.NodeGroups {
		if ng.Count < 0 {
			ng.Count = 0
		}
		if ng.Count > maxInstancesPerMIG {
			ng.Count = maxInstancesPerMIG
		}
	}
	cm.cluster.SetNodeGroups(req.NodeGroups)
	cm.cluster.Spec.Project = req.GceProject
	if cm.cluster.Spec.Project == "" {
		cm.cluster.Spec.Project = cm.cluster.Spec.CloudCredential[credential.GCEProjectID]
	}

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

	cloud.GenClusterTokens(cm.cluster)

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
	cm.cluster.Spec.KubernetesVersion = "v" + req.Version
	return nil
}

func (cm *clusterManager) updateContext() error {
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
	cm.cluster.Spec.ClusterExternalDomain = cm.ctx.Extra().ExternalDomain(cm.cluster.Name)
	cm.cluster.Spec.ClusterInternalDomain = cm.ctx.Extra().InternalDomain(cm.cluster.Name)
	//if cm.ctx.AppsCodeClusterCreator == "" {
	//	cm.ctx.AppsCodeClusterCreator = cm.ctx.Auth.User.UserName
	//}
	cm.cluster.Spec.EnableWebhookTokenAuthentication = true
	cm.cluster.Spec.EnableApiserverBasicAudit = true
	return nil
}

func (cm *clusterManager) LoadDefaultContext() error {
	err := cm.cluster.Spec.KubeEnv.SetDefaults()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	cm.cluster.Spec.ClusterExternalDomain = cm.ctx.Extra().ExternalDomain(cm.cluster.Name)
	cm.cluster.Spec.ClusterInternalDomain = cm.ctx.Extra().InternalDomain(cm.cluster.Name)

	cm.cluster.Status.Phase = api.ClusterPhasePending
	cm.cluster.Spec.OS = "ubuntu"
	cm.cluster.Spec.MasterSKU = "n1-standard-2"

	cm.cluster.Spec.MasterDiskType = "pd-standard" // "pd-ssd"
	cm.cluster.Spec.MasterDiskSize = 100
	cm.cluster.Spec.NodeDiskType = "pd-standard"
	cm.cluster.Spec.NodeDiskSize = 100

	// https://cloud.google.com/compute/docs/containers/container_vms
	// Comes pre installed with Docker and Kubelet
	cm.cluster.Spec.InstanceImage = "ubuntu-1604-xenial-v20170721" //"kube12-tamal"   // "debian-8-jessie-v20160219" // "container-vm-v20151215"
	cm.cluster.Spec.InstanceImageProject = "ubuntu-os-cloud"       //"k8s-dev" // "debian-cloud"              // "google-containers"

	// REGISTER_MASTER_KUBELET = false // always false, keep master lightweight

	// PREEMPTIBLE_NODE = false // Removed Support

	cm.cluster.Spec.MasterReservedIP = "auto" // GCE - change to "" for avoid allocating Elastic IP
	cm.cluster.Spec.MasterIPRange = "10.246.0.0/24"
	cm.cluster.Spec.ClusterIPRange = "10.244.0.0/16"
	cm.cluster.Spec.ServiceClusterIPRange = "10.0.0.0/16"
	cm.cluster.Spec.NodeScopes = []string{"compute-rw", "monitoring", "logging-write", "storage-ro"}
	cm.cluster.Spec.PollSleepInterval = 3

	cm.cluster.Spec.RegisterMasterKubelet = true
	cm.cluster.Spec.EnableNodePublicIP = true // from aws

	//gcs
	cm.cluster.Spec.AllocateNodeCIDRs = true

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

	// TODO(admin): Node autoscaler is always on, make it a choice
	cm.cluster.Spec.EnableNodeAutoscaler = false
	// cm.ctx.AutoscalerMinNodes = 1
	// cm.ctx.AutoscalerMaxNodes = 100
	cm.cluster.Spec.TargetNodeUtilization = 0.7

	cm.cluster.Spec.AdmissionControl = "NamespaceLifecycle,LimitRanger,ServiceAccount,ResourceQuota,PersistentVolumeLabel"
	// KUBE_UP_AUTOMATIC_CLEANUP

	cm.cluster.Spec.NetworkProvider = "none"
	cm.cluster.Spec.HairpinMode = "promiscuous-bridge"
	// cm.ctx.KubeletPort = "10250"

	version, err := semver.NewVersion(cm.cluster.Spec.KubernetesVersion)
	if err != nil {
		version, err = semver.NewVersion(cm.cluster.Spec.KubernetesVersion)
		if err != nil {
			return err
		}
	}
	version = version.ToBuilder().ResetPrerelease().ResetMetadata().Done()

	// https://github.com/appscode/kubernetes/blob/v1.3.6/cluster/gce/config-cloud.sh#L19
	v_1_3, _ := semver.NewConstraint(">= 1.3, < 1.4")
	if v_1_3.Check(version) {
		// Evict pods whenever compute resource availability on the nodes gets below a threshold.
		cm.cluster.Spec.EvictionHard = `memory.available<100Mi`

		cm.cluster.Spec.NetworkProvider = "kubenet"

		// Evict pods whenever compute resource availability on the nodes gets below a threshold.
		cm.cluster.Spec.EvictionHard = `memory.available<100Mi`
	}

	// https://github.com/appscode/kubernetes/blob/1.4.0-ac/cluster/gce/config-cloud.sh#L19
	v_1_4, _ := semver.NewConstraint(">= 1.4")
	if v_1_4.Check(version) {
		cm.cluster.Spec.ClusterIPRange = "10.244.0.0/14"
		cm.cluster.Spec.NetworkProvider = "kubenet"

		// Evict pods whenever compute resource availability on the nodes gets below a threshold.
		cm.cluster.Spec.EvictionHard = `memory.available<100Mi,nodefs.available<10%,nodefs.inodesFree<5%`

		cm.cluster.Spec.AdmissionControl = "NamespaceLifecycle,LimitRanger,ServiceAccount,PersistentVolumeLabel,DefaultStorageClass,ResourceQuota"
		cm.cluster.Spec.EnableRescheduler = true
	}

	cloud.BuildRuntimeConfig(cm.cluster)
	return nil
}

func (cm *clusterManager) UploadStartupConfig() error {
	if _, err := cm.conn.storageService.Buckets.Get(cm.cluster.Spec.BucketName).Do(); err != nil {
		_, err := cm.conn.storageService.Buckets.Insert(cm.cluster.Spec.Project, &bstore.Bucket{
			Name: cm.cluster.Spec.BucketName,
		}).Do()
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cm.ctx.Logger().Debug("Created bucket %s", cm.cluster.Spec.BucketName)
	} else {
		cm.ctx.Logger().Debug("Bucket %s already exists", cm.cluster.Spec.BucketName)
	}

	{
		caData := &bstore.Object{
			Name: "kubernetes/context/" + strconv.FormatInt(cm.cluster.Spec.ResourceVersion, 10) + "/pki/" + "ca.crt",
		}
		caCert, err := base64.StdEncoding.DecodeString(cm.cluster.Spec.CaCert)
		if _, err = cm.conn.storageService.Objects.Insert(cm.cluster.Spec.BucketName, caData).Media(strings.NewReader(string(caCert))).Do(); err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}

		caKeyData := &bstore.Object{
			Name: "kubernetes/context/" + strconv.FormatInt(cm.cluster.Spec.ResourceVersion, 10) + "/pki/" + "ca.key",
		}
		caKey, err := base64.StdEncoding.DecodeString(cm.cluster.Spec.CaKey)

		if _, err = cm.conn.storageService.Objects.Insert(cm.cluster.Spec.BucketName, caKeyData).Media(strings.NewReader(string(caKey))).Do(); err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		frontCAData := &bstore.Object{
			Name: "kubernetes/context/" + strconv.FormatInt(cm.cluster.Spec.ResourceVersion, 10) + "/pki/" + "front-proxy-ca.crt",
		}
		frontCACert, err := base64.StdEncoding.DecodeString(cm.cluster.Spec.FrontProxyCaCert)
		if _, err = cm.conn.storageService.Objects.Insert(cm.cluster.Spec.BucketName, frontCAData).Media(strings.NewReader(string(frontCACert))).Do(); err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		frontCAKeyData := &bstore.Object{
			Name: "kubernetes/context/" + strconv.FormatInt(cm.cluster.Spec.ResourceVersion, 10) + "/pki/" + "front-proxy-ca.key",
		}
		frontCAKey, err := base64.StdEncoding.DecodeString(cm.cluster.Spec.FrontProxyCaKey)
		if _, err = cm.conn.storageService.Objects.Insert(cm.cluster.Spec.BucketName, frontCAKeyData).Media(strings.NewReader(string(frontCAKey))).Do(); err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}

	}
	{
		cfg, err := cm.cluster.StartupConfigResponse(api.RoleKubernetesMaster)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		data := &bstore.Object{
			Name: "kubernetes/context/" + strconv.FormatInt(cm.cluster.Spec.ResourceVersion, 10) + "/startup-config/" + api.RoleKubernetesMaster + ".yaml",
		}
		_, err = cm.conn.storageService.Objects.Insert(cm.cluster.Spec.BucketName, data).Media(strings.NewReader(cfg)).Do()
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
			Name: "kubernetes/context/" + strconv.FormatInt(cm.cluster.Spec.ResourceVersion, 10) + "/startup-config/" + api.RoleKubernetesPool + ".yaml",
		}
		_, err = cm.conn.storageService.Objects.Insert(cm.cluster.Spec.BucketName, data).Media(strings.NewReader(cfg)).Do()
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}
	return nil
}
