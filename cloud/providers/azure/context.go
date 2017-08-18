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

	//cluster.Spec.ctx.Name = req.Name
	//cluster.Spec.ctx.PHID = phid.NewKubeCluster()
	//cluster.Spec.ctx.Provider = req.Provider
	//cluster.Spec.ctx.Zone = req.Zone
	cm.cluster.Spec.Region = cm.cluster.Spec.Zone
	cm.cluster.Spec.DoNotDelete = req.DoNotDelete

	cm.cluster.SetNodeGroups(req.NodeGroups)

	cm.cluster.Spec.KubernetesMasterName = cm.namer.MasterName()
	cm.cluster.Spec.SSHKey, err = api.NewSSHKeyPair()
	if err != nil {
		return errors.FromErr(err).Err()
	}
	cm.cluster.Spec.SSHKeyExternalID = cm.namer.GenSSHKeyExternalID()
	cm.cluster.Spec.SSHKeyPHID = phid.NewSSHKey()

	// cluster.Spec.ctx.MasterSGName = cluster.Spec.ctx.Name + "-master-" + rand.Characters(6)
	// cluster.Spec.ctx.NodeSGName = cluster.Spec.ctx.Name + "-node-" + rand.Characters(6)

	cloud.GenClusterTokens(cm.cluster)

	cm.cluster.Spec.KubeadmToken = cloud.GetKubeadmToken()
	cm.cluster.Spec.KubernetesVersion = "v" + req.Version

	cm.cluster.Spec.AzureCloudConfig = &api.AzureCloudConfig{
		TenantID:           cm.cluster.Spec.CloudCredential[credential.AzureTenantID],
		SubscriptionID:     cm.cluster.Spec.CloudCredential[credential.AzureSubscriptionID],
		AadClientID:        cm.cluster.Spec.CloudCredential[credential.AzureClientID],
		AadClientSecret:    cm.cluster.Spec.CloudCredential[credential.AzureClientSecret],
		ResourceGroup:      cm.namer.ResourceGroupName(),
		Location:           cm.cluster.Spec.Zone,
		SubnetName:         cm.namer.SubnetName(),
		SecurityGroupName:  cm.namer.NetworkSecurityGroupName(),
		VnetName:           cm.namer.VirtualNetworkName(),
		RouteTableName:     cm.namer.RouteTableName(),
		StorageAccountName: cm.namer.GenStorageAccountName(),
	}
	cm.cluster.Spec.CloudConfigPath = "/etc/kubernetes/azure.json"
	cm.cluster.Spec.AzureStorageAccountName = cm.cluster.Spec.AzureCloudConfig.StorageAccountName

	return nil
}

func (cm *clusterManager) LoadDefaultContext() error {
	err := cm.cluster.Spec.KubeEnv.SetDefaults()
	if err != nil {
		return errors.FromErr(err).Err()
	}

	cm.cluster.Spec.ClusterExternalDomain = cm.ctx.Extra().ExternalDomain(cm.cluster.Name)
	cm.cluster.Spec.ClusterInternalDomain = cm.ctx.Extra().InternalDomain(cm.cluster.Name)

	cm.cluster.Status.Phase = api.ClusterPhasePending
	// cm.cluster.Spec.OS = "Debian" // offer: "16.04.0-LTS"

	// https://docs.microsoft.com/en-us/azure/virtual-machines/virtual-machines-windows-sizes#d-series
	cm.cluster.Spec.MasterSKU = "Standard_D2_v2" // CPU 2 Memory 7 disk 4
	cm.cluster.Spec.InstanceRootPassword = rand.GeneratePassword()

	// Disk size can't be set for boot disk
	// cm.cluster.Spec.MasterDiskType = "pd-standard" // "pd-ssd"
	// cm.cluster.Spec.MasterDiskSize = 100
	// cm.cluster.Spec.NodeDiskType = "pd-standard"
	// cm.cluster.Spec.NodeDiskSize = 100

	/*
		"imageReference": {
			"publisher": "credativ",
			"offer": "Debian",
			"sku": "8",
			"version": "latest"
		}
	*/
	//cm.cluster.Spec.InstanceImageProject = "credativ" // publisher
	//cm.cluster.Spec.InstanceImage = "8"               // sku
	//cm.cluster.Spec.InstanceImageVersion = "latest"   // version

	// https://docs.microsoft.com/en-us/azure/virtual-machines/linux/cli-ps-findimage
	// Canonical:UbuntuServer:16.04-LTS:latest @dipta
	cm.cluster.Spec.OS = "UbuntuServer"
	cm.cluster.Spec.InstanceImageProject = "Canonical"
	cm.cluster.Spec.InstanceImage = "16.04-LTS"
	cm.cluster.Spec.InstanceImageVersion = "latest"

	// REGISTER_MASTER_KUBELET = false // always false, keep master lightweight

	// PREEMPTIBLE_NODE = false // Removed Support

	cm.cluster.Spec.MasterReservedIP = "auto" // GCE - change to "" for avoid allocating Elastic IP
	cm.cluster.Spec.MasterIPRange = "10.246.0.0/24"
	cm.cluster.Spec.ClusterIPRange = "10.244.0.0/16"
	cm.cluster.Spec.ServiceClusterIPRange = "10.0.0.0/16"
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

	cm.cluster.Spec.AdmissionControl = "NamespaceLifecycle,LimitRanger,ServiceAccount,PersistentVolumeLabel,DefaultStorageClass,ResourceQuota"
	// KUBE_UP_AUTOMATIC_CLEANUP

	cm.cluster.Spec.VpcCidrBase = "10.240"
	cm.cluster.Spec.MasterIPSuffix = ".4"
	cm.cluster.Spec.MasterInternalIP = cm.cluster.Spec.VpcCidrBase + ".1" + cm.cluster.Spec.MasterIPSuffix // "10.240.1.4"

	//cm.ctx.VpcCidr = cm.ctx.VpcCidrBase + ".0.0/16"
	cm.cluster.Spec.SubnetCidr = "10.240.0.0/16"

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

	v_1_4, _ := semver.NewConstraint(">= 1.4")
	if v_1_4.Check(version) {
		cm.cluster.Spec.NetworkProvider = "kubenet"
		cm.cluster.Spec.AdmissionControl = "NamespaceLifecycle,LimitRanger,ServiceAccount,PersistentVolumeLabel,DefaultStorageClass,ResourceQuota"
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
