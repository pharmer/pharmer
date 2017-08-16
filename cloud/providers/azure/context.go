package azure

import (
	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/context"
	"github.com/appscode/pharmer/credential"
	"github.com/appscode/pharmer/phid"
	"github.com/appscode/pharmer/util/kubeadm"
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
		return errors.FromErr(err).Err()
	}
	cm.namer = namer{cluster: cm.cluster}

	//cluster.ctx.Name = req.Name
	//cluster.ctx.PHID = phid.NewKubeCluster()
	//cluster.ctx.Provider = req.Provider
	//cluster.ctx.Zone = req.Zone
	cm.cluster.Region = cm.cluster.Zone
	cm.cluster.DoNotDelete = req.DoNotDelete

	cm.cluster.SetNodeGroups(req.NodeGroups)

	cm.cluster.KubernetesMasterName = cm.namer.MasterName()
	cm.cluster.SSHKey, err = api.NewSSHKeyPair()
	if err != nil {
		return errors.FromErr(err).Err()
	}
	cm.cluster.SSHKeyExternalID = cm.namer.GenSSHKeyExternalID()
	cm.cluster.SSHKeyPHID = phid.NewSSHKey()

	// cluster.ctx.MasterSGName = cluster.ctx.Name + "-master-" + rand.Characters(6)
	// cluster.ctx.NodeSGName = cluster.ctx.Name + "-node-" + rand.Characters(6)

	cloud.GenClusterTokens(cm.cluster)

	cm.cluster.KubeadmToken = kubeadm.GetRandomToken()
	cm.cluster.KubeVersion = "v" + req.Version

	cm.cluster.AzureCloudConfig = &api.AzureCloudConfig{
		TenantID:           cm.cluster.CloudCredential[credential.AzureTenantID],
		SubscriptionID:     cm.cluster.CloudCredential[credential.AzureSubscriptionID],
		AadClientID:        cm.cluster.CloudCredential[credential.AzureClientID],
		AadClientSecret:    cm.cluster.CloudCredential[credential.AzureClientSecret],
		ResourceGroup:      cm.namer.ResourceGroupName(),
		Location:           cm.cluster.Zone,
		SubnetName:         cm.namer.SubnetName(),
		SecurityGroupName:  cm.namer.NetworkSecurityGroupName(),
		VnetName:           cm.namer.VirtualNetworkName(),
		RouteTableName:     cm.namer.RouteTableName(),
		StorageAccountName: cm.namer.GenStorageAccountName(),
	}
	cm.cluster.CloudConfigPath = "/etc/kubernetes/azure.json"
	cm.cluster.AzureStorageAccountName = cm.cluster.AzureCloudConfig.StorageAccountName

	return nil
}

func (cm *clusterManager) LoadDefaultContext() error {
	err := cm.cluster.KubeEnv.SetDefaults()
	if err != nil {
		return errors.FromErr(err).Err()
	}

	cm.cluster.ClusterExternalDomain = cm.ctx.Extra().ExternalDomain(cm.cluster.Name)
	cm.cluster.ClusterInternalDomain = cm.ctx.Extra().InternalDomain(cm.cluster.Name)

	cm.cluster.Status = api.KubernetesStatus_Pending
	// cm.cluster.OS = "Debian" // offer: "16.04.0-LTS"

	// https://docs.microsoft.com/en-us/azure/virtual-machines/virtual-machines-windows-sizes#d-series
	cm.cluster.MasterSKU = "Standard_D2_v2" // CPU 2 Memory 7 disk 4
	cm.cluster.InstanceRootPassword = rand.GeneratePassword()

	cm.cluster.AppsCodeLogIndexPrefix = "logstash-"
	cm.cluster.AppsCodeLogStorageLifetime = 90 * 24 * 3600
	cm.cluster.AppsCodeMonitoringStorageLifetime = 90 * 24 * 3600

	// Disk size can't be set for boot disk
	// cm.cluster.MasterDiskType = "pd-standard" // "pd-ssd"
	// cm.cluster.MasterDiskSize = 100
	// cm.cluster.NodeDiskType = "pd-standard"
	// cm.cluster.NodeDiskSize = 100

	/*
		"imageReference": {
			"publisher": "credativ",
			"offer": "Debian",
			"sku": "8",
			"version": "latest"
		}
	*/
	//cm.cluster.InstanceImageProject = "credativ" // publisher
	//cm.cluster.InstanceImage = "8"               // sku
	//cm.cluster.InstanceImageVersion = "latest"   // version

	// https://docs.microsoft.com/en-us/azure/virtual-machines/linux/cli-ps-findimage
	// Canonical:UbuntuServer:16.04-LTS:latest @dipta
	cm.cluster.OS = "UbuntuServer"
	cm.cluster.InstanceImageProject = "Canonical"
	cm.cluster.InstanceImage = "16.04-LTS"
	cm.cluster.InstanceImageVersion = "latest"

	// REGISTER_MASTER_KUBELET = false // always false, keep master lightweight

	// PREEMPTIBLE_NODE = false // Removed Support

	cm.cluster.MasterReservedIP = "auto" // GCE - change to "" for avoid allocating Elastic IP
	cm.cluster.MasterIPRange = "10.246.0.0/24"
	cm.cluster.ClusterIPRange = "10.244.0.0/16"
	cm.cluster.ServiceClusterIPRange = "10.0.0.0/16"
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

	cm.cluster.AdmissionControl = "NamespaceLifecycle,LimitRanger,ServiceAccount,PersistentVolumeLabel,DefaultStorageClass,ResourceQuota"
	// KUBE_UP_AUTOMATIC_CLEANUP

	cm.cluster.VpcCidrBase = "10.240"
	cm.cluster.MasterIPSuffix = ".4"
	cm.cluster.MasterInternalIP = cm.cluster.VpcCidrBase + ".1" + cm.cluster.MasterIPSuffix // "10.240.1.4"

	//cm.ctx.VpcCidr = cm.ctx.VpcCidrBase + ".0.0/16"
	cm.cluster.SubnetCidr = "10.240.0.0/16"

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

	v_1_4, _ := semver.NewConstraint(">= 1.4")
	if v_1_4.Check(version) {
		cm.cluster.NetworkProvider = "kubenet"
		cm.cluster.AdmissionControl = "NamespaceLifecycle,LimitRanger,ServiceAccount,PersistentVolumeLabel,DefaultStorageClass,ResourceQuota"
	}

	cloud.BuildRuntimeConfig(cm.cluster)
	return nil
}

func (cm *clusterManager) UploadStartupConfig() error {
	if api.UseFirebase() {
		return cloud.UploadAllCertsInFirebase(cm.ctx, cm.cluster)
	}
	return nil
}
