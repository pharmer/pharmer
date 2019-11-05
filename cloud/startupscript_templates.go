/*
Copyright The Pharmer Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cloud

import (
	"text/template"

	api "pharmer.dev/pharmer/apis/v1alpha1"
)

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

curl -fsSL --retry 5 -o pre-k {{ .PrekVersion }} \
	&& chmod +x pre-k \
	&& mv pre-k /usr/bin/


timedatectl set-timezone Etc/UTC
{{ template "prepare-host" . }}
{{ template "mount-master-pd" . }}


rm -rf /usr/sbin/policy-rc.d
systemctl enable docker kubelet nfs-utils
systemctl start docker kubelet nfs-utils

kubeadm reset -f

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

{{ if eq .NetworkProvider "flannel" }}
{{ template "flannel" . }}
{{ else if eq .NetworkProvider "calico" }}
{{ template "calico" . }}
{{ else if eq .NetworkProvider "weavenet" }}
{{ template "weavenet" . }}
{{ end }}

kubectl apply \
  -f https://raw.githubusercontent.com/pharmer/addons/release-1.13.1/kubeadm-probe/installer.yaml \
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

curl -fsSL --retry 5 -o pre-k {{ .PrekVersion }} \
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


kubeadm reset -f
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
cmd='kubectl apply --kubeconfig /etc/kubernetes/admin.conf -f https://raw.githubusercontent.com/pharmer/addons/release-1.13.1/cloud-controller-manager/rbac.yaml'
exec_until_success "$cmd"

# Deploy CCM DaemonSet
cmd='kubectl apply --kubeconfig /etc/kubernetes/admin.conf -f https://raw.githubusercontent.com/pharmer/addons/release-1.13.1/cloud-controller-manager/{{ .Provider }}/installer.yaml'
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
kubectl apply \
  -f https://docs.projectcalico.org/v3.3/getting-started/kubernetes/installation/hosted/rbac-kdd.yaml \
  --kubeconfig /etc/kubernetes/admin.conf

kubectl apply \
  -f https://docs.projectcalico.org/v3.3/getting-started/kubernetes/installation/hosted/kubernetes-datastore/calico-networking/1.7/calico.yaml \
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
  -f https://raw.githubusercontent.com/pharmer/addons/release-1.13.1/flannel/v0.9.1/kube-vxlan.yml \
  --kubeconfig /etc/kubernetes/admin.conf
`))
)
