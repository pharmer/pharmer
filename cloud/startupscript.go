package cloud

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"text/template"

	"github.com/ghodss/yaml"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pkg/errors"
	version "gomodules.xyz/version"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
)

// https://github.com/pharmer/pharmer/issues/347
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

func (td TemplateData) InitConfigurationYAML() (string, error) {
	if td.InitConfiguration == nil {
		return "", nil
	}
	var cb []byte
	var err error

	if td.IsVersionLessThan1_13() {
		conf := ConvertInitConfigFromV1bet1ToV1alpha3(td.InitConfiguration)
		cb, err = yaml.Marshal(conf)
	} else {
		cb, err = yaml.Marshal(td.InitConfiguration)
	}

	return string(cb), err
}

func (td TemplateData) ClusterConfigurationYAML() (string, error) {
	if td.ClusterConfiguration == nil {
		return "", nil
	}
	var cb []byte
	var err error
	if td.IsVersionLessThan1_13() {
		conf := ConvertClusterConfigFromV1beta1ToV1alpha3(td.ClusterConfiguration)
		cb, err = yaml.Marshal(conf)
	} else {
		cb, err = yaml.Marshal(td.ClusterConfiguration)
	}
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
	if td.IsVersionLessThan1_13() {
		apiAddress := strings.Split(td.APIServerAddress, ":")
		if len(apiAddress) < 2 {
			return "", errors.Errorf("Apiserver address is not correct")
		}
		apiPort, err := strconv.Atoi(apiAddress[1])
		if err != nil {
			return "", err
		}
		conf := ConvertJoinConfigFromV1beta1ToV1alpha3(&cfg)
		conf.ClusterName = td.ClusterName
		conf.APIEndpoint.AdvertiseAddress = apiAddress[0]
		conf.APIEndpoint.BindPort = int32(apiPort)
		cb, err = yaml.Marshal(conf)
	}
	return string(cb), err
}

func (td TemplateData) ForceKubeadmResetFlag() (string, error) {
	lv11 := td.IsVersionLessThan("1.11.0")
	if !lv11 {
		return "-f", nil
	}
	return "", nil
}

func (td TemplateData) IsVersionLessThan(currentVersion string) bool {
	cv, _ := version.NewVersion(td.KubernetesVersion)
	v11, _ := version.NewVersion(currentVersion)
	return cv.LessThan(v11)
}

func (td TemplateData) IsVersionLessThan1_13() bool {
	return td.IsVersionLessThan("1.13.0")
}

func (td TemplateData) IsKubeadmV1Alpha3() bool {
	return !td.IsVersionLessThan("1.12.0")
}

func (td TemplateData) IsVersionLessThan1_11() bool {
	return td.IsVersionLessThan("1.11.0")
}

func (td TemplateData) UseKubeProxy1_11_0() bool {
	v, _ := version.NewVersion(td.KubernetesVersion)
	if v.ToMutator().Version.String() == "1.11.0" {
		return true
	}
	return false
}

// Forked kubeadm 1.8.x for: https://github.com/kubernetes/kubernetes/pull/49840
func (td TemplateData) UseForkedKubeadm_1_8_3() bool {
	v, _ := version.NewVersion(td.KubernetesVersion)
	return v.ToMutator().ResetPrerelease().ResetMetadata().ResetPatch().String() == "1.8.0"
}

func (td TemplateData) KubeletExtraArgsStr() string {
	var buf bytes.Buffer
	for k, v := range td.KubeletExtraArgs {
		buf.WriteString("--")
		buf.WriteString(k)
		buf.WriteRune('=')
		buf.WriteString(v)
		buf.WriteRune(' ')
	}
	return buf.String()
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

func (td *TemplateData) ControlPlaneEndpointsFromLB(cfg *kubeadmapi.ClusterConfiguration, cluster *api.Cluster) {
	if cluster.Status.Cloud.LoadBalancer.DNS != "" {
		cfg.ControlPlaneEndpoint = fmt.Sprintf("%s:%d", cluster.Status.Cloud.LoadBalancer.DNS, cluster.Status.Cloud.LoadBalancer.Port)
		cfg.APIServer.CertSANs = append(cfg.APIServer.CertSANs, cluster.Status.Cloud.LoadBalancer.DNS)
	} else if cluster.Status.Cloud.LoadBalancer.IP != "" {
		cfg.ControlPlaneEndpoint = fmt.Sprintf("%s:%d", cluster.Status.Cloud.LoadBalancer.IP, cluster.Status.Cloud.LoadBalancer.Port)
		cfg.APIServer.CertSANs = append(cfg.APIServer.CertSANs, cluster.Status.Cloud.LoadBalancer.IP)
	}
}

var (
	StartupScriptTemplate = template.Must(template.New(api.RoleMaster).Parse(`
{{- template "init-script" }}

# kill apt processes (E: Unable to lock directory /var/lib/apt/lists/)
kill $(ps aux | grep '[a]pt' | awk '{print $2}') || true

{{ template "init-os" . }}

# https://major.io/2016/05/05/preventing-ubuntu-16-04-starting-daemons-package-installed/
echo -e '#!/bin/bash\nexit 101' > /usr/sbin/policy-rc.d
chmod +x /usr/sbin/policy-rc.d

{{- template "install-docker-script" }}

apt-get install -y apt-transport-https curl ca-certificates software-properties-common tzdata
curl -fsSL --retry 5 https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
echo 'deb http://apt.kubernetes.io/ kubernetes-xenial main' > /etc/apt/sources.list.d/kubernetes.list
exec_until_success 'add-apt-repository -y ppa:gluster/glusterfs-3.10'
apt-get update -y
exec_until_success 'apt-get install -y {{ .PackageList }}'
{{ if .UseForkedKubeadm_1_8_3 }}
curl -fsSL --retry 5 -o kubeadm	https://github.com/appscode/kubernetes/releases/download/v1.8.3/kubeadm \
	&& chmod +x kubeadm \
	&& mv kubeadm /usr/bin/
{{ end }}

curl -fsSL --retry 5 -o pre-k https://cdn.appscode.com/binaries/pre-k/{{ .PrekVersion }}/pre-k-linux-amd64 \
	&& chmod +x pre-k \
	&& mv pre-k /usr/bin/

timedatectl set-timezone Etc/UTC
{{ template "prepare-host" . }}
{{ template "mount-master-pd" . }}


rm -rf /usr/sbin/policy-rc.d
systemctl enable docker kubelet nfs-utils
systemctl start docker kubelet nfs-utils

kubeadm reset {{ .ForceKubeadmResetFlag }}

{{ template "setup-certs" . }}

{{ template "cloud-config" . }}

mkdir -p /etc/kubernetes/kubeadm



{{ template "pre-k" . }}

{{ if not .ControlPlaneJoin }}
kubeadm init --config=/etc/kubernetes/kubeadm/config.yaml --skip-token-print
{{ else }}

cat > /etc/kubernetes/kubeadm/join.yaml <<EOF
{{ .JoinConfiguration }}
EOF

PUBLIC_IP=$(pre-k machine public-ips --all=false)
echo $PUBLIC_IP
sed -i "s/CONTROLPLANEIP/$PUBLIC_IP/g" /etc/kubernetes/kubeadm/join.yaml
cat /etc/kubernetes/kubeadm/join.yaml

kubeadm join --config=/etc/kubernetes/kubeadm/join.yaml

{{ end }}

{{ if .UseKubeProxy1_11_0 }}
kubectl apply -f https://raw.githubusercontent.com/pharmer/addons/release-1.11/kube-proxy/v1.11.0/kube-proxy.yaml \
  --kubeconfig /etc/kubernetes/admin.conf
{{ end }}

{{ if eq .NetworkProvider "flannel" }}
{{ template "flannel" . }}
{{ else if eq .NetworkProvider "calico" }}
{{ template "calico" . }}
{{ else if eq .NetworkProvider "weavenet" }}
{{ template "weavenet" . }}
{{ end }}

kubectl apply \
  -f https://raw.githubusercontent.com/pharmer/addons/release-1.11/kubeadm-probe/installer.yaml \
  --kubeconfig /etc/kubernetes/admin.conf

mkdir -p ~/.kube
sudo cp -i /etc/kubernetes/admin.conf ~/.kube/config
sudo chown $(id -u):$(id -g) ~/.kube/config

kubectl apply \
  -f https://raw.githubusercontent.com/pharmer/addons/clusterapi/cluster-api/cluster-crd.yaml \
  --kubeconfig /etc/kubernetes/admin.conf

{{ if .ExternalProvider }}
{{ template "ccm" . }}
{{ template "install-storage-plugin" . }}
{{end}}

{{ template "prepare-cluster" . }}
`))

	_ = template.Must(StartupScriptTemplate.New(api.RoleNode).Parse(`
{{- template "init-script" }}

# kill apt processes (E: Unable to lock directory /var/lib/apt/lists/)
kill $(ps aux | grep '[a]pt' | awk '{print $2}') || true

{{ template "init-os" . }}

# https://major.io/2016/05/05/preventing-ubuntu-16-04-starting-daemons-package-installed/
echo -e '#!/bin/bash\nexit 101' > /usr/sbin/policy-rc.d
chmod +x /usr/sbin/policy-rc.d

{{- template "install-docker-script" }}

apt-get install -y apt-transport-https curl ca-certificates software-properties-common tzdata
curl -fsSL --retry 5 https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
echo 'deb http://apt.kubernetes.io/ kubernetes-xenial main' > /etc/apt/sources.list.d/kubernetes.list
exec_until_success 'add-apt-repository -y ppa:gluster/glusterfs-3.10'
apt-get update -y
exec_until_success 'apt-get install -y {{ .PackageList }}'
{{ if .UseForkedKubeadm_1_8_3 }}
curl -fsSL --retry 5 -o kubeadm	https://github.com/appscode/kubernetes/releases/download/v1.8.3/kubeadm \
	&& chmod +x kubeadm \
	&& mv kubeadm /usr/bin/
{{ end }}
curl -fsSL --retry 5 -o pre-k https://cdn.appscode.com/binaries/pre-k/{{ .PrekVersion }}/pre-k-linux-amd64 \
	&& chmod +x pre-k \
	&& mv pre-k /usr/bin/

timedatectl set-timezone Etc/UTC
{{ template "prepare-host" . }}

systemctl daemon-reload
rm -rf /usr/sbin/policy-rc.d
systemctl enable docker kubelet nfs-utils
systemctl start docker kubelet nfs-utils

mkdir -p /etc/kubernetes/kubeadm

cat > /etc/kubernetes/kubeadm/join.yaml <<EOF
{{ .JoinConfiguration }}
EOF


kubeadm reset {{ .ForceKubeadmResetFlag }}
kubeadm join --config=/etc/kubernetes/kubeadm/join.yaml
`))

	_ = template.Must(StartupScriptTemplate.New("init-script").Parse(`#!/bin/bash
set -euxo pipefail
# log to /var/log/pharmer.log
exec > >(tee -a /var/log/pharmer.log)
exec 2>&1

export DEBIAN_FRONTEND=noninteractive
export DEBCONF_NONINTERACTIVE_SEEN=true

exec_until_success() {
	$1
	while [ $? -ne 0 ]; do
		sleep 2
		$1
	done
}
`))

	_ = template.Must(StartupScriptTemplate.New("init-os").Parse(``))

	_ = template.Must(StartupScriptTemplate.New("cloud-config").Parse(``))

	_ = template.Must(StartupScriptTemplate.New("prepare-host").Parse(``))

	_ = template.Must(StartupScriptTemplate.New("mount-master-pd").Parse(``))

	_ = template.Must(StartupScriptTemplate.New("prepare-cluster").Parse(``))
	_ = template.Must(StartupScriptTemplate.New("install-docker-script").Parse(`
apt-get update -y
apt-get install -y \
    apt-transport-https \
    gnupg-agent

curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -

sudo apt-key fingerprint 0EBFCD88

sudo add-apt-repository \
   "deb [arch=amd64] https://download.docker.com/linux/ubuntu \
   $(lsb_release -cs) \
   stable"

apt-get update -y

apt-get install -y docker-ce=18.06.1~ce~3-0~ubuntu
`))

	_ = template.Must(StartupScriptTemplate.New("setup-certs").Parse(`
mkdir -p /etc/kubernetes/pki

cat > /etc/kubernetes/pki/ca.key <<EOF
{{ .CAKey }}
EOF
pre-k get ca-cert --common-name=ca < /etc/kubernetes/pki/ca.key > /etc/kubernetes/pki/ca.crt

cat > /etc/kubernetes/pki/front-proxy-ca.key <<EOF
{{ .FrontProxyKey }}
EOF
pre-k get ca-cert --common-name=front-proxy-ca < /etc/kubernetes/pki/front-proxy-ca.key > /etc/kubernetes/pki/front-proxy-ca.crt
chmod 600 /etc/kubernetes/pki/ca.key /etc/kubernetes/pki/front-proxy-ca.key

cat > /etc/kubernetes/pki/sa.key <<EOF
{{ .SAKey }}
EOF
pre-k get pub-key < /etc/kubernetes/pki/sa.key > /etc/kubernetes/pki/sa.pub

# ETCD Keys
mkdir -p /etc/kubernetes/pki/etcd
cat > /etc/kubernetes/pki/etcd/ca.key <<EOF
{{ .ETCDCAKey }}
EOF
pre-k get ca-cert --common-name=kubernetes < /etc/kubernetes/pki/etcd/ca.key > /etc/kubernetes/pki/etcd/ca.crt
chmod 600 /etc/kubernetes/pki/etcd/ca.key

`))

	_ = template.Must(StartupScriptTemplate.New("ccm").Parse(`
# Deploy CCM RBAC
cmd='kubectl apply --kubeconfig /etc/kubernetes/admin.conf -f https://raw.githubusercontent.com/pharmer/addons/master/cloud-controller-manager/rbac.yaml'
exec_until_success "$cmd"

# Deploy CCM DaemonSet
cmd='kubectl apply --kubeconfig /etc/kubernetes/admin.conf -f https://raw.githubusercontent.com/pharmer/addons/master/cloud-controller-manager/{{ .Provider }}/installer.yaml'
exec_until_success "$cmd"

until [ $(kubectl get pods -n kube-system -l k8s-app=kube-dns -o jsonpath='{.items[0].status.phase}' --kubeconfig /etc/kubernetes/admin.conf) == "Running" ]
do
   echo '.'
   sleep 5
done
`))
	_ = template.Must(StartupScriptTemplate.New("install-storage-plugin").Parse(``))

	_ = template.Must(StartupScriptTemplate.New("pre-k").Parse(`
{{ if .InitConfiguration }}
cat > /etc/kubernetes/kubeadm/init.yaml <<EOF
{{ .InitConfigurationYAML }}
EOF
{{ end }}

{{ if .ClusterConfiguration }}
cat > /etc/kubernetes/kubeadm/cluster.yaml <<EOF
{{ .ClusterConfigurationYAML }}
EOF
{{ end }}

pre-k merge config \
	--init-config=/etc/kubernetes/kubeadm/init.yaml \
    --cluster-config=/etc/kubernetes/kubeadm/cluster.yaml \
	--apiserver-advertise-address=$(pre-k machine public-ips --all=false) \
	--apiserver-cert-extra-sans=$(pre-k machine public-ips --routable) \
	--apiserver-cert-extra-sans=$(pre-k machine private-ips) \
	--node-name=${NODE_NAME:-} \
	> /etc/kubernetes/kubeadm/config.yaml	

`))
	_ = template.Must(StartupScriptTemplate.New("calico").Parse(`
{{ if .IsVersionLessThan1_11 }}
kubectl apply \
  -f https://raw.githubusercontent.com/pharmer/addons/master/calico/2.6/calico.yaml \
  --kubeconfig /etc/kubernetes/admin.conf
{{ else }}
kubectl apply \
  -f https://docs.projectcalico.org/v3.3/getting-started/kubernetes/installation/hosted/rbac-kdd.yaml \
  --kubeconfig /etc/kubernetes/admin.conf

kubectl apply \
  -f https://docs.projectcalico.org/v3.3/getting-started/kubernetes/installation/hosted/kubernetes-datastore/calico-networking/1.7/calico.yaml \
  --kubeconfig /etc/kubernetes/admin.conf
{{ end }}
`))

	_ = template.Must(StartupScriptTemplate.New("weavenet").Parse(`
sysctl net.bridge.bridge-nf-call-iptables=1
export kubever=$(kubectl version --kubeconfig /etc/kubernetes/admin.conf | base64 | tr -d '\n')
kubectl apply \
  -f "https://cloud.weave.works/k8s/net?k8s-version=$kubever" \
  --kubeconfig /etc/kubernetes/admin.conf
`))

	_ = template.Must(StartupScriptTemplate.New("flannel").Parse(`
kubectl apply \
  -f https://raw.githubusercontent.com/pharmer/addons/release-1.11/flannel/v0.9.1/kube-vxlan.yml \
  --kubeconfig /etc/kubernetes/admin.conf
`))
)
