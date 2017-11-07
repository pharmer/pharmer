package cloud

import (
	"bytes"
	"strings"
	"text/template"

	api "github.com/appscode/pharmer/apis/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/hashicorp/go-version"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1alpha1"
)

type TemplateData struct {
	ClusterName      string
	KubeletVersion   string
	KubeadmVersion   string
	KubeadmToken     string
	CAKey            string
	FrontProxyKey    string
	APIServerAddress string
	ExtraDomains     string
	NetworkProvider  string
	CloudConfig      string
	Provider         string
	ExternalProvider bool

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

func (td TemplateData) IsPreReleaseVersion() bool {
	if v, err := version.NewVersion(td.KubeadmVersion); err == nil && v.Prerelease() != "" {
		return true
	}
	return false
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

func (td TemplateData) KubeletExtraArgsEmptyCloudProviderStr() string {
	var buf bytes.Buffer
	for k, v := range td.KubeletExtraArgs {
		if k == "cloud-config" {
			continue
		}
		if k == "cloud-provider" {
			v = ""
		}
		buf.WriteString("--")
		buf.WriteString(k)
		buf.WriteRune('=')
		buf.WriteString(v)
		buf.WriteRune(' ')
	}
	return buf.String()
}

func (td TemplateData) PackageList() string {
	pkgs := []string{
		"cron",
		"docker.io",
		"ebtables",
		"git",
		"glusterfs-client",
		"haveged",
		"nfs-common",
		"socat",
	}
	if !td.IsPreReleaseVersion() {
		if td.KubeletVersion == "" {
			pkgs = append(pkgs, "kubelet", "kubectl")
		} else {
			pkgs = append(pkgs, "kubelet="+td.KubeletVersion+"*", "kubectl="+td.KubeletVersion+"*")
		}
		if td.KubeadmVersion == "" {
			pkgs = append(pkgs, "kubeadm")
		} else {
			pkgs = append(pkgs, "kubeadm="+td.KubeadmVersion+"*")
		}
	}
	if td.Provider != "gce" && td.Provider != "gke" {
		pkgs = append(pkgs, "ntp")
	}
	return strings.Join(pkgs, " ")
}

var (
	StartupScriptTemplate = template.Must(template.New(api.RoleMaster).Parse(`#!/bin/bash
set -euxo pipefail
# log to /var/log/pharmer.log
exec > >(tee -a /var/log/pharmer.log)
exec 2>&1

export DEBIAN_FRONTEND=noninteractive
export DEBCONF_NONINTERACTIVE_SEEN=true

# kill apt processes (E: Unable to lock directory /var/lib/apt/lists/)
kill $(ps aux | grep '[a]pt' | awk '{print $2}') || true

{{ template "init-os" . }}
{{ template "init-script" }}

# https://major.io/2016/05/05/preventing-ubuntu-16-04-starting-daemons-package-installed/
echo -e '#!/bin/bash\nexit 101' > /usr/sbin/policy-rc.d
chmod +x /usr/sbin/policy-rc.d

apt-get update -y
apt-get install -y apt-transport-https curl ca-certificates software-properties-common tzdata
curl -fsSL --retry 5 https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
echo 'deb http://apt.kubernetes.io/ kubernetes-xenial main' > /etc/apt/sources.list.d/kubernetes.list
exec-until-success 'add-apt-repository -y ppa:gluster/glusterfs-3.10'
apt-get update -y
apt-get install -y {{ .PackageList }} || true
{{ if .IsPreReleaseVersion }}
curl -fsSL --retry 5 -o kubeadm https://dl.k8s.io/release/{{ .KubeadmVersion }}/bin/linux/amd64/kubeadm \
    && chmod +x kubeadm \
	&& mv kubeadm /usr/bin/
{{ else if not .ExternalProvider }}
curl -fsSL --retry 5 -o kubeadm	https://github.com/appscode/kubernetes/releases/download/v1.8.3/kubeadm \
	&& chmod +x kubeadm \
	&& mv kubeadm /usr/bin/
{{ end }}

curl -fsSL --retry 5 -o pre-k https://cdn.appscode.com/binaries/pre-k/0.1.0-alpha.10/pre-k-linux-amd64 \
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

mkdir -p /etc/kubernetes/pharmer
{{ if .CloudConfig }}
cat > /etc/kubernetes/pharmer/cloud-config <<EOF
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
	--apiserver-advertise-address=$(pre-k get public-ips --all=false) \
	--apiserver-cert-extra-sans=$(pre-k get public-ips --routable) \
	--apiserver-cert-extra-sans=$(pre-k get private-ips) \
	--apiserver-cert-extra-sans={{ .ExtraDomains }} \
	> /etc/kubernetes/kubeadm/config.yaml
kubeadm init --config=/etc/kubernetes/kubeadm/config.yaml --skip-token-print

{{ if eq .NetworkProvider "flannel" }}
{{ template "flannel" . }}
{{ else if eq .NetworkProvider "calico" }}
{{ template "calico" . }}
{{ end }}

kubectl apply \
  -f https://raw.githubusercontent.com/appscode/pharmer/master/addons/kubeadm-probe/installer.yaml \
  --kubeconfig /etc/kubernetes/admin.conf

mkdir -p ~/.kube
sudo cp -i /etc/kubernetes/admin.conf ~/.kube/config
sudo chown $(id -u):$(id -g) ~/.kube/config

{{ if .ExternalProvider }}
{{ template "ccm" . }}
{{end}}

{{ template "prepare-cluster" . }}
`))

	_ = template.Must(StartupScriptTemplate.New(api.RoleNode).Parse(`#!/bin/bash
set -euxo pipefail
# log to /var/log/pharmer.log
exec > >(tee -a /var/log/pharmer.log)
exec 2>&1

export DEBIAN_FRONTEND=noninteractive
export DEBCONF_NONINTERACTIVE_SEEN=true

# kill apt processes (E: Unable to lock directory /var/lib/apt/lists/)
kill $(ps aux | grep '[a]pt' | awk '{print $2}') || true

{{ template "init-os" . }}
{{ template "init-script" }}

# https://major.io/2016/05/05/preventing-ubuntu-16-04-starting-daemons-package-installed/
echo -e '#!/bin/bash\nexit 101' > /usr/sbin/policy-rc.d
chmod +x /usr/sbin/policy-rc.d

apt-get update -y
apt-get install -y apt-transport-https curl ca-certificates software-properties-common tzdata
curl -fsSL --retry 5 https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
echo 'deb http://apt.kubernetes.io/ kubernetes-xenial main' > /etc/apt/sources.list.d/kubernetes.list
exec-until-success 'add-apt-repository -y ppa:gluster/glusterfs-3.10'
apt-get update -y
apt-get install -y {{ .PackageList }} || true
{{ if .IsPreReleaseVersion }}
curl -fsSL --retry 5 -o kubeadm https://dl.k8s.io/release/{{ .KubeadmVersion }}/bin/linux/amd64/kubeadm \
    && chmod +x kubeadm \
	&& mv kubeadm /usr/bin/
{{ else if not .ExternalProvider }}
curl -fsSL --retry 5 -o kubeadm	https://github.com/appscode/kubernetes/releases/download/v1.8.3/kubeadm \
	&& chmod +x kubeadm \
	&& mv kubeadm /usr/bin/
{{ end }}
curl -fsSL --retry 5 -o pre-k https://cdn.appscode.com/binaries/pre-k/0.1.0-alpha.10/pre-k-linux-amd64 \
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
kubeadm join --token={{ .KubeadmToken }} {{ .APIServerAddress }}
`))

	_ = template.Must(StartupScriptTemplate.New("init-os").Parse(``))

	_ = template.Must(StartupScriptTemplate.New("init-script").Parse(`
function exec-until-success() {
	$1
	while [ $? -ne 0 ]; do
		sleep 2
		$1
	done
}
`))

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
cmd='kubectl apply --kubeconfig /etc/kubernetes/admin.conf -f https://raw.githubusercontent.com/appscode/pharmer/master/addons/cloud-controller-manager/rbac.yaml'
exec-until-success "$cmd"

# Deploy CCM DaemonSet
cmd='kubectl apply --kubeconfig /etc/kubernetes/admin.conf -f https://raw.githubusercontent.com/appscode/pharmer/master/addons/cloud-controller-manager/{{ .Provider }}/installer.yaml'
exec-until-success "$cmd"

until [ $(kubectl get pods -n kube-system -l k8s-app=kube-dns -o jsonpath='{.items[0].status.phase}' --kubeconfig /etc/kubernetes/admin.conf) == "Running" ]
do
   echo '.'
   sleep 5
done

# kubectl taint nodes $(uname -n) node.cloudprovider.kubernetes.io/uninitialized=true:NoSchedule --kubeconfig /etc/kubernetes/admin.conf
#
# cat > /etc/systemd/system/kubelet.service.d/20-pharmer.conf <<EOF
# [Service]
# Environment="KUBELET_EXTRA_ARGS={{ .KubeletExtraArgsStr }}"
# EOF
# systemctl daemon-reload
# systemctl restart kubelet
# systemctl restart docker
`))
	_ = template.Must(StartupScriptTemplate.New("calico").Parse(`
kubectl apply \
  -f https://raw.githubusercontent.com/appscode/pharmer/master/addons/calico/2.6/calico.yaml \
  --kubeconfig /etc/kubernetes/admin.conf
`))

	_ = template.Must(StartupScriptTemplate.New("flannel").Parse(`
kubectl apply \
  -f https://raw.githubusercontent.com/coreos/flannel/v0.8.0/Documentation/kube-flannel.yml \
  --kubeconfig /etc/kubernetes/admin.conf
kubectl apply \
  -f https://raw.githubusercontent.com/coreos/flannel/v0.8.0/Documentation/kube-flannel-rbac.yml \
  --kubeconfig /etc/kubernetes/admin.conf
`))
)
