package cloud

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"text/template"

	"github.com/ghodss/yaml"
	"github.com/hashicorp/go-version"
	api "github.com/pharmer/pharmer/apis/v1alpha1"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1alpha1"
)

// https://github.com/pharmer/pharmer/issues/347
var kubernetesCNIVersions = map[string]string{
	"1.8.0": "0.5.1",
	"1.9.0": "0.6.0",
}

var prekVersions = map[string]string{
	"1.8.0": "1.8.0",
	"1.9.0": "1.9.0-rc.0",
}

type TemplateData struct {
	ClusterName       string
	KubernetesVersion string
	KubeadmToken      string
	CloudCredential   map[string]string
	CAHash            string
	CAKey             string
	FrontProxyKey     string
	APIServerAddress  string
	NetworkProvider   string
	CloudConfig       string
	Provider          string
	NodeName          string
	ExternalProvider  bool

	MasterConfiguration *kubeadmapi.MasterConfiguration
	KubeletExtraArgs    map[string]string
}

func (td TemplateData) MasterConfigurationYAML() (string, error) {
	if td.MasterConfiguration == nil {
		return "", nil
	}
	cb, err := yaml.Marshal(td.MasterConfiguration)
	return string(cb), err
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

	pkgs := []string{
		"cron",
		"docker.io",
		"ebtables",
		"git",
		"glusterfs-client",
		"haveged",
		"jq",
		"nfs-common",
		"socat",
		"kubelet=" + patch + "*",
		"kubectl=" + patch + "*",
		"kubeadm=" + patch + "*",
	}
	if cni, found := kubernetesCNIVersions[minor]; !found {
		return "", fmt.Errorf("kubernetes-cni version is unknown for Kubernetes version %s", td.KubernetesVersion)
	} else {
		pkgs = append(pkgs, "kubernetes-cni="+cni+"*")
	}

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
		return "", fmt.Errorf("pre-k version is unknown for Kubernetes version %s", td.KubernetesVersion)
	}
	return prekVer, nil
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

apt-get update -y
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

cat > /etc/systemd/system/kubelet.service.d/20-pharmer.conf <<EOF
[Service]
Environment="KUBELET_EXTRA_ARGS={{ .KubeletExtraArgsStr }}"
EOF
systemctl daemon-reload
rm -rf /usr/sbin/policy-rc.d
systemctl enable docker kubelet nfs-utils
systemctl start docker kubelet nfs-utils

kubeadm reset

{{ template "setup-certs" . }}

mkdir -p /etc/kubernetes/ccm
{{ if .CloudConfig }}
cat > /etc/kubernetes/ccm/cloud-config <<EOF
{{ .CloudConfig }}
EOF
{{ end }}

mkdir -p /etc/kubernetes/kubeadm

{{ if .MasterConfiguration }}
cat > /etc/kubernetes/kubeadm/base.yaml <<EOF
{{ .MasterConfigurationYAML }}
EOF
{{ end }}

pre-k merge master-config \
	--config=/etc/kubernetes/kubeadm/base.yaml \
	--apiserver-advertise-address=$(pre-k machine public-ips --all=false) \
	--apiserver-cert-extra-sans=$(pre-k machine public-ips --routable) \
	--apiserver-cert-extra-sans=$(pre-k machine private-ips) \
	--node-name=${NODE_NAME:-} \
	> /etc/kubernetes/kubeadm/config.yaml
kubeadm init --config=/etc/kubernetes/kubeadm/config.yaml --skip-token-print

{{ if eq .NetworkProvider "flannel" }}
{{ template "flannel" . }}
{{ else if eq .NetworkProvider "calico" }}
{{ template "calico" . }}
{{ else if eq .NetworkProvider "weavenet" }}
{{ template "weavenet" . }}
{{ end }}

kubectl apply \
  -f https://raw.githubusercontent.com/pharmer/addons/master/kubeadm-probe/installer.yaml \
  --kubeconfig /etc/kubernetes/admin.conf

mkdir -p ~/.kube
sudo cp -i /etc/kubernetes/admin.conf ~/.kube/config
sudo chown $(id -u):$(id -g) ~/.kube/config

{{ if .ExternalProvider }}
{{ template "ccm" . }}
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

apt-get update -y
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

cat > /etc/systemd/system/kubelet.service.d/20-pharmer.conf <<EOF
[Service]
Environment="KUBELET_EXTRA_ARGS={{ .KubeletExtraArgsStr }}"
EOF
systemctl daemon-reload
rm -rf /usr/sbin/policy-rc.d
systemctl enable docker kubelet nfs-utils
systemctl start docker kubelet nfs-utils

{{ if not .ExternalProvider }}
{{ if .CloudConfig }}
cat > /etc/kubernetes/cloud-config <<EOF
{{ .CloudConfig }}
EOF
{{ end }}
{{ end }}

kubeadm reset
kubeadm join --token={{ .KubeadmToken }} --discovery-token-ca-cert-hash={{ .CAHash }} {{ .APIServerAddress }}
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

	_ = template.Must(StartupScriptTemplate.New("prepare-host").Parse(``))

	_ = template.Must(StartupScriptTemplate.New("prepare-cluster").Parse(``))

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

	_ = template.Must(StartupScriptTemplate.New("calico").Parse(`
kubectl apply \
  -f https://raw.githubusercontent.com/pharmer/addons/master/calico/2.6/calico.yaml \
  --kubeconfig /etc/kubernetes/admin.conf
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
  -f https://raw.githubusercontent.com/pharmer/addons/master/flannel/v0.9.1/kube-vxlan.yml \
  --kubeconfig /etc/kubernetes/admin.conf
`))
)
