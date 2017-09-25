package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"text/template"

	"github.com/appscode/pharmer/api"
	"github.com/ghodss/yaml"
	"github.com/hashicorp/go-version"
	"gopkg.in/ini.v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/cert"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1alpha1"
)

type TemplateData struct {
	IsPreReleaseVersion bool
	KubernetesVersion   string
	KubeadmVersion      string
	KubeadmToken        string
	CAKey               string
	FrontProxyKey       string
	APIServerHost       string
	ExtraDomains        string
	NetworkProvider     string
	MasterConfiguration string
	CloudConfigPath     string
	CloudConfig         string
	NodeGroupName       string
}

func GetTemplateData(ctx context.Context, cluster *api.Cluster, nodeGroup string) TemplateData {
	td := TemplateData{
		KubernetesVersion: cluster.Spec.KubernetesVersion,
		KubeadmVersion:    cluster.Spec.KubeadmVersion,
		KubeadmToken:      cluster.Spec.Token,
		CAKey:             string(cert.EncodePrivateKeyPEM(CAKey(ctx))),
		FrontProxyKey:     string(cert.EncodePrivateKeyPEM(FrontProxyCAKey(ctx))),
		APIServerHost:     cluster.APIServerHost(),
		ExtraDomains:      cluster.Spec.ClusterExternalDomain,
		NetworkProvider:   cluster.Spec.Networking.NetworkProvider,
		NodeGroupName:     nodeGroup,
	}
	if cluster.Spec.KubeadmVersion != "" {
		if v, err := version.NewVersion(cluster.Spec.KubeadmVersion); err == nil && v.Prerelease() != "" {
			td.IsPreReleaseVersion = true
		}
	}

	cfg := kubeadmapi.MasterConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubeadm.k8s.io/v1alpha1",
			Kind:       "MasterConfiguration",
		},
		API:  cluster.Spec.API,
		Etcd: cluster.Spec.Etcd,
		Networking: kubeadmapi.Networking{
			ServiceSubnet: cluster.Spec.Networking.ServiceSubnet,
			PodSubnet:     cluster.Spec.Networking.PodSubnet,
			DNSDomain:     cluster.Spec.Networking.DNSDomain,
		},
		KubernetesVersion: cluster.Spec.KubernetesVersion,
		CloudProvider:     cluster.Spec.Cloud.CloudProvider,
		// AuthorizationModes:
		Token:                      cluster.Spec.Token,
		TokenTTL:                   cluster.Spec.TokenTTL,
		SelfHosted:                 cluster.Spec.SelfHosted,
		APIServerExtraArgs:         map[string]string{},
		ControllerManagerExtraArgs: map[string]string{},
		SchedulerExtraArgs:         map[string]string{},
		APIServerCertSANs:          []string{},
	}
	{
		if cluster.Spec.Cloud.GCE != nil {
			cfg.APIServerExtraArgs["cloud-config"] = cluster.Spec.Cloud.CloudConfigPath
			td.CloudConfigPath = cluster.Spec.Cloud.CloudConfigPath
			// ref: https://github.com/kubernetes/kubernetes/blob/release-1.5/cluster/gce/configure-vm.sh#L846
			cfg := ini.Empty()
			err := cfg.Section("global").ReflectFrom(cluster.Spec.Cloud.GCE.CloudConfig)
			if err != nil {
				panic(err)
			}
			var buf bytes.Buffer
			_, err = cfg.WriteTo(&buf)
			if err != nil {
				panic(err)
			}
			td.CloudConfig = buf.String()
		}
	}
	{
		if cluster.Spec.Cloud.Azure != nil {
			cfg.APIServerExtraArgs["cloud-config"] = cluster.Spec.Cloud.CloudConfigPath
			td.CloudConfigPath = cluster.Spec.Cloud.CloudConfigPath

			data, err := json.MarshalIndent(cluster.Spec.Cloud.Azure.CloudConfig, "", "  ")
			if err != nil {
				panic(err)
			}
			td.CloudConfig = string(data)
		}
	}
	{
		extraDomains := []string{}
		if domain := Extra(ctx).ExternalDomain(cluster.Name); domain != "" {
			extraDomains = append(extraDomains, domain)
		}
		if domain := Extra(ctx).InternalDomain(cluster.Name); domain != "" {
			extraDomains = append(extraDomains, domain)
		}
		td.ExtraDomains = strings.Join(extraDomains, ",")
	}
	cb, err := yaml.Marshal(&cfg)
	if err != nil {
		panic(err)
	}
	td.MasterConfiguration = string(cb)
	return td
}

func RenderStartupScript(ctx context.Context, cluster *api.Cluster, role, nodeGroup string) (string, error) {
	var buf bytes.Buffer
	if err := StartupScriptTemplate.ExecuteTemplate(&buf, role, GetTemplateData(ctx, cluster, nodeGroup)); err != nil {
		return "", err
	}
	return buf.String(), nil
}

var (
	StartupScriptTemplate = template.Must(template.New(api.RoleMaster).Parse(`#!/bin/bash
set -x
set -o errexit
set -o nounset
set -o pipefail

# log to /var/log/startup-script.log
exec > >(tee -a /var/log/startup-script.log)
exec 2>&1

# kill apt processes (E: Unable to lock directory /var/lib/apt/lists/)
kill $(ps aux | grep '[a]pt' | awk '{print $2}') || true

{{ template "prepare-host" . }}

apt-get update -y
apt-get install -y apt-transport-https curl ca-certificates

curl -fSsL https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
echo 'deb http://apt.kubernetes.io/ kubernetes-xenial main' > /etc/apt/sources.list.d/kubernetes.list

add-apt-repository -y ppa:gluster/glusterfs-3.10

apt-get update -y
apt-get install -y \
	socat \
	ebtables \
	git \
	haveged \
	nfs-common \
	cron \
	glusterfs-client \
	kubelet \
	{{ if .IsPreReleaseVersion }}kubeadm{{ if .KubeadmVersion }}={{ .KubeadmVersion }}{{ end }} \
	cloud-utils \
	docker.io || true

{{ if .IsPreReleaseVersion }}
wget -O kubeadm dl.k8s.io/release/{{ .KubeadmVersion }}/bin/linux/amd64/kubeadm \
    && chmod +x kubeadm \
	&& sudo mv kubeadm /usr/bin/
{{ end }}

curl -Lo pre-k https://cdn.appscode.com/binaries/pre-k/0.1.0-alpha.3/pre-k-linux-amd64 \
	&& chmod +x pre-k \
	&& mv pre-k /usr/bin/

systemctl enable docker
systemctl start docker

kubeadm reset

{{ template "setup-certs" . }}

{{ if .CloudConfigPath }}
cat > {{ .CloudConfigPath }} <<EOF
{{ .CloudConfig }}
EOF
{{ end }}

mkdir -p /etc/kubernetes/kubeadm

{{ if .MasterConfiguration }}
cat > /etc/kubernetes/kubeadm/config.yaml <<EOF
{{ .MasterConfiguration }}
EOF
{{ end }}

pre-k merge master-config \
	--config=/etc/kubernetes/kubeadm/config.yaml \
	--apiserver-bind-port=6443 \
	--token={{ .KubeadmToken }} \
	--apiserver-advertise-address=$(pre-k get public-ips --all=false) \
	--apiserver-cert-extra-sans=$(pre-k get public-ips --routable) \
	--apiserver-cert-extra-sans=$(pre-k get private-ips) \
	--apiserver-cert-extra-sans={{ .ExtraDomains }} \
	--kubernetes-version={{ .KubernetesVersion }} \
	> /etc/kubernetes/kubeadm/config.yaml
kubeadm init --config=/etc/kubernetes/kubeadm/config.yaml --skip-token-print

{{ if eq .NetworkProvider "flannel" }}
{{ template "flannel" . }}
{{ else if eq .NetworkProvider "calico" }}
{{ template "calico" . }}
{{ end }}

mkdir -p ~/.kube
sudo cp -i /etc/kubernetes/admin.conf ~/.kube/config
sudo chown $(id -u):$(id -g) ~/.kube/config
`))

	_ = template.Must(StartupScriptTemplate.New(api.RoleNode).Parse(`#!/bin/bash
set -x
set -o errexit
set -o nounset
set -o pipefail

# log to /var/log/startup-script.log
exec > >(tee -a /var/log/startup-script.log)
exec 2>&1

# kill apt processes (E: Unable to lock directory /var/lib/apt/lists/)
kill $(ps aux | grep '[a]pt' | awk '{print $2}') || true

{{ template "prepare-host" . }}

apt-get update -y
apt-get install -y apt-transport-https curl ca-certificates

curl -fSsL https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
echo 'deb http://apt.kubernetes.io/ kubernetes-xenial main' > /etc/apt/sources.list.d/kubernetes.list

add-apt-repository -y ppa:gluster/glusterfs-3.10

apt-get update -y
apt-get install -y \
	socat \
	ebtables \
	git \
	haveged \
	nfs-common \
	cron \
	glusterfs-client \
	kubelet \
	{{ if .IsPreReleaseVersion }}kubeadm{{ if .KubeadmVersion }}={{ .KubeadmVersion }}{{ end }} \
	docker.io || true

{{ if .IsPreReleaseVersion }}
wget -O kubeadm dl.k8s.io/release/{{ .KubeadmVersion }}/bin/linux/amd64/kubeadm \
    && chmod +x kubeadm \
	&& sudo mv kubeadm /usr/bin/
{{ end }}

systemctl enable docker
systemctl start docker

cat > /etc/systemd/system/kubelet.service.d/20-label-taints.conf <<EOF
[Service]
Environment="KUBELET_EXTRA_ARGS=--node-labels=cloud.appscode.com/pool={{ .NodeGroupName }}"
EOF
systemctl daemon-reload
systemctl restart kubelet

kubeadm reset
kubeadm join --token={{ .KubeadmToken }} {{ .APIServerHost }}:6443
`))

	_ = template.Must(StartupScriptTemplate.New("prepare-host").Parse(``))

	_ = template.Must(StartupScriptTemplate.New("setup-certs").Parse(`
mkdir -p /etc/kubernetes/pki

cat > /etc/kubernetes/pki/ca.key <<EOF
{{ .CAKey }}
EOF
pre-k get cacert --common-name=ca < /etc/kubernetes/pki/ca.key > /etc/kubernetes/pki/ca.crt

cat > /etc/kubernetes/pki/front-proxy-ca.key <<EOF
{{ .FrontProxyKey }}
EOF
pre-k get cacert --common-name=front-proxy-ca < /etc/kubernetes/pki/front-proxy-ca.key > /etc/kubernetes/pki/front-proxy-ca.crt

chmod 600 /etc/kubernetes/pki/ca.key /etc/kubernetes/pki/front-proxy-ca.key
`))

	_ = template.Must(StartupScriptTemplate.New("calico").Parse(`
kubectl apply \
  -f http://docs.projectcalico.org/v2.3/getting-started/kubernetes/installation/hosted/kubeadm/1.6/calico.yaml \
  --kubeconfig /etc/kubernetes/admin.conf
`))

	_ = template.Must(StartupScriptTemplate.New("flannel").Parse(`
kubectl apply \
  -f https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml \
  --kubeconfig /etc/kubernetes/admin.conf

kubectl apply \
  -f https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel-rbac.yml \
  --kubeconfig /etc/kubernetes/admin.conf
`))
)
