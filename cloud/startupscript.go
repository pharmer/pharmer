package cloud

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"gomodules.xyz/cert"
	"gomodules.xyz/version"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/pubkeypin"
	api "pharmer.dev/pharmer/apis/v1alpha1"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/util"
)

// https://pharmer.dev/pharmer/issues/347
var kubernetesCNIVersions = map[string]string{
	"1.8.0":  "0.5.1",
	"1.9.0":  "0.6.0",
	"1.10.0": "0.6.0",
	"1.11.0": "0.6.0",
	"1.12.0": "0.6.0",
	"1.13.0": "0.6.0",
	"1.13.5": "0.7.5",
	"1.13.6": "0.7.5",
	"1.14.0": "0.7.5",
}

var prekVersions = map[string]string{
	"1.8.0":  "1.8.0",
	"1.9.0":  "1.9.0",
	"1.10.0": "1.10.0",
	"1.11.0": "1.12.0-alpha.3",
	"1.12.0": "1.12.0-alpha.3",
	"1.13.0": "1.13.0",
	"1.14.0": "1.13.0",
}

type TemplateData struct {
	ClusterName       string
	KubernetesVersion string
	KubeadmToken      string
	CloudCredential   map[string]string
	CAHash            string
	CAKey             string
	FrontProxyKey     string
	SAKey             string
	ETCDCAKey         string
	APIServerAddress  string
	NetworkProvider   string
	CloudConfig       string
	Provider          string
	NodeName          string
	ExternalProvider  bool

	InitConfiguration    *kubeadmapi.InitConfiguration
	ClusterConfiguration *kubeadmapi.ClusterConfiguration
	JoinConfiguration    string
	KubeletExtraArgs     map[string]string
	ControlPlaneJoin     bool
}

func NewNodeTemplateData(cm Interface, machine *clusterv1.Machine, token string) TemplateData {
	certs := cm.GetCertificates()
	cluster := cm.GetCluster()

	td := TemplateData{
		ClusterName:       cluster.Name,
		KubeadmToken:      token,
		KubernetesVersion: cluster.Spec.Config.KubernetesVersion,
		CAHash:            pubkeypin.Hash(certs.CACert.Cert),
		CAKey:             string(cert.EncodePrivateKeyPEM(certs.CACert.Key)),
		FrontProxyKey:     string(cert.EncodePrivateKeyPEM(certs.FrontProxyCACert.Key)),
		SAKey:             string(cert.EncodePrivateKeyPEM(certs.ServiceAccountCert.Key)),
		ETCDCAKey:         string(cert.EncodePrivateKeyPEM(certs.EtcdCACert.Key)),
		APIServerAddress:  cluster.APIServerAddress(),
		NetworkProvider:   cluster.ClusterConfig().Cloud.NetworkProvider,
		Provider:          cluster.ClusterConfig().Cloud.CloudProvider,
	}

	td.KubeletExtraArgs = cluster.ClusterConfig().KubeletExtraArgs

	if td.KubeletExtraArgs == nil {
		td.KubeletExtraArgs = make(map[string]string)
	}

	td.KubeletExtraArgs["node-labels"] = api.NodeLabels{
		api.NodePoolKey: machine.Name,
		api.RoleNodeKey: "",
	}.String()

	return cm.NewNodeTemplateData(machine, token, td)
}

func NewMasterTemplateData(cm Interface, machine *clusterv1.Machine, token string) TemplateData {
	td := NewNodeTemplateData(cm, machine, "")
	td.KubeletExtraArgs["node-labels"] = api.NodeLabels{
		api.NodePoolKey: machine.Name,
	}.String()

	td.InitConfiguration = &kubeadmapi.InitConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubeadm.k8s.io/v1beta1",
			Kind:       "InitConfiguration",
		},

		NodeRegistration: kubeadmapi.NodeRegistrationOptions{
			KubeletExtraArgs: td.KubeletExtraArgs,
		},
		LocalAPIEndpoint: kubeadmapi.APIEndpoint{
			BindPort: 6443,
		},
	}

	if token != "" {
		td.ControlPlaneJoin = true

		joinConf, err := td.JoinConfigurationYAML()
		if err != nil {
			panic(err)
		}
		td.JoinConfiguration = joinConf
	}

	return cm.NewMasterTemplateData(machine, token, td)
}

func RenderStartupScript(cm Interface, machine *clusterapi.Machine, token, customTemplate string) (string, error) {
	tpl, err := StartupScriptTemplate.Clone()
	if err != nil {
		return "", err
	}
	tpl, err = tpl.Parse(customTemplate)
	if err != nil {
		return "", err
	}

	var script bytes.Buffer
	if util.IsControlPlaneMachine(machine) {
		if err := tpl.ExecuteTemplate(&script, api.RoleMaster, NewMasterTemplateData(cm, machine, token)); err != nil {
			return "", err
		}
	} else {
		if err := tpl.ExecuteTemplate(&script, api.RoleNode, NewNodeTemplateData(cm, machine, token)); err != nil {
			return "", err
		}
	}
	return script.String(), nil
}

func GetDefaultKubeadmClusterConfig(cluster *api.Cluster, hostPath *kubeadmapi.HostPathMount) *kubeadmapi.ClusterConfiguration {
	cfg := &kubeadmapi.ClusterConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubeadm.k8s.io/v1beta1",
			Kind:       "ClusterConfiguration",
		},
		Networking: kubeadmapi.Networking{
			ServiceSubnet: cluster.Spec.ClusterAPI.Spec.ClusterNetwork.Services.CIDRBlocks[0],
			PodSubnet:     cluster.Spec.ClusterAPI.Spec.ClusterNetwork.Pods.CIDRBlocks[0],
			DNSDomain:     cluster.Spec.ClusterAPI.Spec.ClusterNetwork.ServiceDomain,
		},
		KubernetesVersion: cluster.ClusterConfig().KubernetesVersion,
		APIServer: kubeadmapi.APIServer{
			ControlPlaneComponent: kubeadmapi.ControlPlaneComponent{
				ExtraArgs: cluster.Spec.Config.APIServerExtraArgs,
			},
			CertSANs: cluster.ClusterConfig().APIServerCertSANs,
		},
		ControllerManager: kubeadmapi.ControlPlaneComponent{
			ExtraArgs: cluster.ClusterConfig().ControllerManagerExtraArgs,
		},
		Scheduler: kubeadmapi.ControlPlaneComponent{
			ExtraArgs: cluster.ClusterConfig().SchedulerExtraArgs,
		},
		ClusterName: cluster.Name,
	}

	if hostPath != nil {
		cfg.APIServer.ExtraVolumes = []kubeadmapi.HostPathMount{*hostPath}
		cfg.ControllerManager.ExtraVolumes = []kubeadmapi.HostPathMount{*hostPath}
	}

	controlPlaneEndpointsFromLB(cfg, cluster)

	return cfg
}

func (td TemplateData) InitConfigurationYAML() (string, error) {
	if td.InitConfiguration == nil {
		return "", nil
	}
	cb, err := yaml.Marshal(td.InitConfiguration)

	return string(cb), err
}

func (td TemplateData) ClusterConfigurationYAML() (string, error) {
	if td.ClusterConfiguration == nil {
		return "", nil
	}
	cb, err := yaml.Marshal(td.ClusterConfiguration)
	return string(cb), err
}

func (td TemplateData) JoinConfigurationYAML() (string, error) {
	var cb []byte

	cfg := kubeadmapi.JoinConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubeadm.k8s.io/v1beta1",
			Kind:       "JoinConfiguration",
		},
		NodeRegistration: kubeadmapi.NodeRegistrationOptions{
			KubeletExtraArgs: td.KubeletExtraArgs,
		},
		Discovery: kubeadmapi.Discovery{
			BootstrapToken: &kubeadmapi.BootstrapTokenDiscovery{
				Token:             td.KubeadmToken,
				APIServerEndpoint: td.APIServerAddress,
				CACertHashes:      []string{td.CAHash},
			},
		},
	}

	if td.ControlPlaneJoin {
		// TODO FIX
		cfg.ControlPlane = &kubeadmapi.JoinControlPlane{}
		cfg.ControlPlane.LocalAPIEndpoint.AdvertiseAddress = "CONTROLPLANEIP"
		cfg.ControlPlane.LocalAPIEndpoint.BindPort = kubeadmapi.DefaultAPIBindPort
	}

	cb, err := yaml.Marshal(cfg)
	return string(cb), err
}

func (td TemplateData) IsVersionLessThan(currentVersion string) bool {
	cv, _ := version.NewVersion(td.KubernetesVersion)
	v11, _ := version.NewVersion(currentVersion)
	return cv.LessThan(v11)
}

func (td TemplateData) PackageList() (string, error) {
	v, err := version.NewVersion(td.KubernetesVersion)
	if err != nil {
		return "", err
	}
	if v.Prerelease() != "" {
		return "", errors.New("pre-release versions are not supported")
	}
	patch := v.Clone().ToMutator().ResetMetadata().ResetPrerelease().String()
	minor := v.Clone().ToMutator().ResetMetadata().ResetPrerelease().ResetPatch().String()
	kubeadmVersion := patch
	if td.IsVersionLessThan("1.12.0") {
		kubeadmVersion = "1.12.0"
	}

	pkgs := []string{
		"cron",
		"ebtables",
		"git",
		"glusterfs-client",
		"haveged",
		"jq",
		"nfs-common",
		"socat",
		"kubelet=" + patch + "*",
		"kubectl=" + patch + "*",
		"kubeadm=" + kubeadmVersion + "*",
	}
	cni, found := kubernetesCNIVersions[patch]
	if !found {
		if cni, found = kubernetesCNIVersions[minor]; !found {
			return "", errors.Errorf("kubernetes-cni version is unknown for Kubernetes version %s", td.KubernetesVersion)
		}
	}
	pkgs = append(pkgs, "kubernetes-cni="+cni+"*")

	if td.Provider != "gce" && td.Provider != "gke" {
		pkgs = append(pkgs, "ntp")
	}
	return strings.Join(pkgs, " "), nil
}

func (td TemplateData) PrekVersion() (string, error) {
	v, err := version.NewVersion(td.KubernetesVersion)
	if err != nil {
		return "", err
	}
	if v.Prerelease() != "" {
		return "", errors.New("pre-release versions are not supported")
	}
	minor := v.ToMutator().ResetMetadata().ResetPrerelease().ResetPatch().String()

	prekVer, found := prekVersions[minor]
	if !found {
		return "", errors.Errorf("pre-k version is unknown for Kubernetes version %s", td.KubernetesVersion)
	}
	return prekVer, nil
}

func controlPlaneEndpointsFromLB(cfg *kubeadmapi.ClusterConfiguration, cluster *api.Cluster) {
	if cluster.Status.Cloud.LoadBalancer.DNS != "" {
		cfg.ControlPlaneEndpoint = fmt.Sprintf("%s:%d", cluster.Status.Cloud.LoadBalancer.DNS, cluster.Status.Cloud.LoadBalancer.Port)
		cfg.APIServer.CertSANs = append(cfg.APIServer.CertSANs, cluster.Status.Cloud.LoadBalancer.DNS)
	} else if cluster.Status.Cloud.LoadBalancer.IP != "" {
		cfg.ControlPlaneEndpoint = fmt.Sprintf("%s:%d", cluster.Status.Cloud.LoadBalancer.IP, cluster.Status.Cloud.LoadBalancer.Port)
		cfg.APIServer.CertSANs = append(cfg.APIServer.CertSANs, cluster.Status.Cloud.LoadBalancer.IP)
	}
}
