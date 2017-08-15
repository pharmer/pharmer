package azure

import (
	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud/lib"
	"github.com/appscode/pharmer/credential"
	"github.com/appscode/pharmer/phid"
	"github.com/appscode/pharmer/storage"
	semver "github.com/hashicorp/go-version"
)

type clusterManager struct {
	ctx   *api.Cluster
	ins   *api.ClusterInstances
	conn  *cloudConnector
	namer namer
}

func (cm *clusterManager) initContext(req *proto.ClusterCreateRequest) error {
	err := cm.LoadDefaultContext()
	if err != nil {
		return errors.FromErr(err).Err()
	}
	cm.namer = namer{ctx: cm.ctx}

	//cluster.ctx.Name = req.Name
	//cluster.ctx.PHID = phid.NewKubeCluster()
	//cluster.ctx.Provider = req.Provider
	//cluster.ctx.Zone = req.Zone
	cm.ctx.Region = cm.ctx.Zone
	cm.ctx.DoNotDelete = req.DoNotDelete
	lib.SetApps(cm.ctx)

	cm.ctx.SetNodeGroups(req.NodeGroups)

	cm.ctx.KubernetesMasterName = cm.namer.MasterName()
	cm.ctx.SSHKey, err = api.NewSSHKeyPair()
	if err != nil {
		return errors.FromErr(err).Err()
	}
	cm.ctx.SSHKeyExternalID = cm.namer.GenSSHKeyExternalID()
	cm.ctx.SSHKeyPHID = phid.NewSSHKey()

	// cluster.ctx.MasterSGName = cluster.ctx.Name + "-master-" + rand.Characters(6)
	// cluster.ctx.NodeSGName = cluster.ctx.Name + "-node-" + rand.Characters(6)

	lib.GenClusterTokens(cm.ctx)

	cm.ctx.AzureCloudConfig = &api.AzureCloudConfig{
		TenantID:           cm.ctx.CloudCredential[credential.AzureTenantID],
		SubscriptionID:     cm.ctx.CloudCredential[credential.AzureSubscriptionID],
		AadClientID:        cm.ctx.CloudCredential[credential.AzureClientID],
		AadClientSecret:    cm.ctx.CloudCredential[credential.AzureClientSecret],
		ResourceGroup:      cm.namer.ResourceGroupName(),
		Location:           cm.ctx.Zone,
		SubnetName:         cm.namer.SubnetName(),
		SecurityGroupName:  cm.namer.NetworkSecurityGroupName(),
		VnetName:           cm.namer.VirtualNetworkName(),
		RouteTableName:     cm.namer.RouteTableName(),
		StorageAccountName: cm.namer.GenStorageAccountName(),
	}
	cm.ctx.CloudConfigPath = "/etc/kubernetes/azure.json"
	cm.ctx.AzureStorageAccountName = cm.ctx.AzureCloudConfig.StorageAccountName

	return nil
}

func (cm *clusterManager) LoadDefaultContext() error {
	err := cm.ctx.KubeEnv.SetDefaults()
	if err != nil {
		return errors.FromErr(err).Err()
	}

	cm.ctx.ClusterExternalDomain = cm.ctx.Extra().ExternalDomain(cm.ctx.Name)
	cm.ctx.ClusterInternalDomain = cm.ctx.Extra().InternalDomain(cm.ctx.Name)

	cm.ctx.Status = storage.KubernetesStatus_Pending
	cm.ctx.OS = "Debian" // offer: "16.04.0-LTS"
	// https://docs.microsoft.com/en-us/azure/virtual-machines/virtual-machines-windows-sizes#d-series
	cm.ctx.MasterSKU = "Standard_D2_v2" // CPU 2 Memory 7 disk 4
	cm.ctx.InstanceRootPassword = rand.GeneratePassword()

	cm.ctx.AppsCodeLogIndexPrefix = "logstash-"
	cm.ctx.AppsCodeLogStorageLifetime = 90 * 24 * 3600
	cm.ctx.AppsCodeMonitoringStorageLifetime = 90 * 24 * 3600

	// Disk size can't be set for boot disk
	// cm.ctx.MasterDiskType = "pd-standard" // "pd-ssd"
	// cm.ctx.MasterDiskSize = 100
	// cm.ctx.NodeDiskType = "pd-standard"
	// cm.ctx.NodeDiskSize = 100

	/*
		"imageReference": {
			"publisher": "credativ",
			"offer": "Debian",
			"sku": "8",
			"version": "latest"
		}
	*/
	cm.ctx.InstanceImageProject = "credativ" // publisher
	cm.ctx.InstanceImage = "8"               // sku
	cm.ctx.InstanceImageVersion = "latest"   // version

	// REGISTER_MASTER_KUBELET = false // always false, keep master lightweight

	// PREEMPTIBLE_NODE = false // Removed Support

	cm.ctx.MasterReservedIP = "auto" // GCE - change to "" for avoid allocating Elastic IP
	cm.ctx.MasterIPRange = "10.246.0.0/24"
	cm.ctx.ClusterIPRange = "10.244.0.0/16"
	cm.ctx.ServiceClusterIPRange = "10.0.0.0/16"
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

	cm.ctx.AdmissionControl = "NamespaceLifecycle,LimitRanger,ServiceAccount,PersistentVolumeLabel,DefaultStorageClass,ResourceQuota"
	// KUBE_UP_AUTOMATIC_CLEANUP

	cm.ctx.VpcCidrBase = "10.240"
	cm.ctx.MasterIPSuffix = ".4"
	cm.ctx.MasterInternalIP = cm.ctx.VpcCidrBase + ".1" + cm.ctx.MasterIPSuffix // "10.240.1.4"

	//cm.ctx.VpcCidr = cm.ctx.VpcCidrBase + ".0.0/16"
	cm.ctx.SubnetCidr = "10.240.0.0/16"

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

	v_1_4, _ := semver.NewConstraint(">= 1.4")
	if v_1_4.Check(version) {
		cm.ctx.NetworkProvider = "kubenet"
		cm.ctx.AdmissionControl = "NamespaceLifecycle,LimitRanger,ServiceAccount,PersistentVolumeLabel,DefaultStorageClass,ResourceQuota"
	}

	lib.BuildRuntimeConfig(cm.ctx)
	return nil
}

func (cm *clusterManager) UploadStartupConfig() error {
	if api.UseFirebase() {
		return lib.UploadStartupConfigInFirebase(cm.ctx)
	}
	return nil
}
