package cloud

import (
	"text/template"

	api "github.com/appscode/pharmer/apis/v1alpha1"
	"github.com/ghodss/yaml"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1alpha1"
)

type TemplateData struct {
	IsPreReleaseVersion bool
	KubernetesVersion   string
	KubeadmVersion      string
	KubeadmToken        string
	CAKey               string
	FrontProxyKey       string
	APIServerAddress    string
	APIBindPort         int32
	ExtraDomains        string
	NetworkProvider     string
	MasterConfiguration *kubeadmapi.MasterConfiguration
	CloudConfigPath     string
	CloudConfig         string
	NodeGroupName       string
	Provider            string
	ExternalProvider    bool
	ConfigurationBucket string

	KubeletExtraArgs map[string]string
}

func (td TemplateData) MasterConfigurationYAML() (string, error) {
	if td.MasterConfiguration == nil {
		return "", nil
	}
	cb, err := yaml.Marshal(td.MasterConfiguration)
	return string(cb), err
}

/*
func GetTemplateData(ctx context.Context, cluster *api.Cluster, token, nodeGroup string, externalProvider bool) TemplateData {
	td := TemplateData{
		KubernetesVersion: cluster.Spec.KubernetesVersion,
		KubeadmVersion:    cluster.Spec.MasterKubeadmVersion,
		KubeadmToken:      token,
		CAKey:             string(cert.EncodePrivateKeyPEM(CAKey(ctx))),
		FrontProxyKey:     string(cert.EncodePrivateKeyPEM(FrontProxyCAKey(ctx))),
		APIServerAddress:  cluster.APIServerAddress(),
		APIBindPort:       6443,
		ExtraDomains:      cluster.Spec.ClusterExternalDomain,
		NetworkProvider:   cluster.Spec.Networking.NetworkProvider,
		NodeGroupName:     nodeGroup,
		Provider:          cluster.Spec.Cloud.CloudProvider,
		ExternalProvider:  externalProvider,
	}
	if cluster.Spec.MasterKubeadmVersion != "" {
		if v, err := version.NewVersion(cluster.Spec.MasterKubeadmVersion); err == nil && v.Prerelease() != "" {
			td.IsPreReleaseVersion = true
		} else {
			if lv, err := GetLatestKubeadmVerson(); err == nil && lv == cluster.Spec.MasterKubeadmVersion {
				td.KubeadmVersion = ""
			}
		}
	}

	{
		if cluster.Spec.Cloud.GCE != nil {
			td.ConfigurationBucket = fmt.Sprintf(`gsutil cat gs://%v/config.sh > /etc/kubernetes/config.sh
			`, cluster.Status.Cloud.GCE.BucketName)
		} else if cluster.Spec.Cloud.AWS != nil {
			td.ConfigurationBucket = fmt.Sprintf(`apt-get install awscli -y
			aws s3api get-object --bucket %v --key config.sh /etc/kubernetes/config.sh`,
				cluster.Status.Cloud.AWS.BucketName)
		}
	}

	cfg := kubeadmapi.MasterConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubeadm.k8s.io/v1alpha1",
			Kind:       "MasterConfiguration",
		},
		API: kubeadmapi.API{
			AdvertiseAddress: cluster.Spec.API.AdvertiseAddress,
			BindPort:         cluster.Spec.API.BindPort,
		},
		Networking: kubeadmapi.Networking{
			ServiceSubnet: cluster.Spec.Networking.ServiceSubnet,
			PodSubnet:     cluster.Spec.Networking.PodSubnet,
			DNSDomain:     cluster.Spec.Networking.DNSDomain,
		},
		KubernetesVersion: cluster.Spec.KubernetesVersion,
		CloudProvider:     cluster.Spec.Cloud.CloudProvider,
		// AuthorizationModes:
		//Token: token,
		//	TokenTTL:                   cluster.Spec.TokenTTL,
		APIServerExtraArgs:         map[string]string{},
		ControllerManagerExtraArgs: map[string]string{},
		SchedulerExtraArgs:         map[string]string{},
		APIServerCertSANs:          []string{},
	}
	if externalProvider {
		cfg.CloudProvider = "external"
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

func KubeConfigScript(kubeadmToken string) (string, error) {
	var buf bytes.Buffer
	var token = struct {
		Token string
	}{Token: kubeadmToken}
	if err := kubConfigScriptTemplate.ExecuteTemplate(&buf, "config", token); err != nil {
		return "", err
	}
	return buf.String(), nil
}

var (
	kubConfigScriptTemplate = template.Must(template.New("config").Parse(`#!/bin/bash
	declare -x KUBEADM_TOKEN={{ .Token }}
	`))
)

func RenderStartupScript(ctx context.Context, cluster *api.Cluster, token, role, nodeGroup string, externalProvider bool) (string, error) {
	var buf bytes.Buffer
	if err := StartupScriptTemplate.ExecuteTemplate(&buf, role, GetTemplateData(ctx, cluster, token, nodeGroup, externalProvider)); err != nil {
		return "", err
	}
	return buf.String(), nil
}
*/

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
	{{ if not .IsPreReleaseVersion }}kubeadm{{ if .KubeadmVersion }}={{ .KubeadmVersion }}{{ end }}{{ end }} \
	cloud-utils \
	docker.io || true

{{ if .IsPreReleaseVersion }}
curl -Lo kubeadm https://dl.k8s.io/release/{{ .KubeadmVersion }}/bin/linux/amd64/kubeadm \
    && chmod +x kubeadm \
	&& mv kubeadm /usr/bin/
{{ end }}

curl -Lo pre-k https://cdn.appscode.com/binaries/pre-k/0.1.0-alpha.5/pre-k-linux-amd64 \
	&& chmod +x pre-k \
	&& mv pre-k /usr/bin/

systemctl enable docker
systemctl start docker

cat > /etc/systemd/system/kubelet.service.d/20-pharmer.conf <<EOF
[Service]
Environment="KUBELET_EXTRA_ARGS=--node-labels=cloud.appscode.com/pool={{ .NodeGroupName }} {{ if  .CloudConfigPath }} --cloud-provider={{ .Provider }} --cloud-config={{ .CloudConfigPath }} {{ end }}"
EOF

systemctl daemon-reload
systemctl restart kubelet

kubeadm reset

{{ template "setup-certs" . }}

{{ if .CloudConfigPath }}
cat > {{ .CloudConfigPath }} <<EOF
{{ .CloudConfig }}
EOF
{{ end }}

mkdir -p /etc/kubernetes/kubeadm

{{ if .MasterConfigurationYAML }}
cat > /etc/kubernetes/kubeadm/config.yaml <<EOF
{{ .MasterConfigurationYAML }}
EOF
{{ end }}

pre-k merge master-config \
	--config=/etc/kubernetes/kubeadm/config.yaml \
	--apiserver-bind-port={{ .APIBindPort }} \
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

{{ if .ExternalProvider }}
{{ template "ccm" . }}
{{end}}
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
	{{ if not .IsPreReleaseVersion }}kubeadm{{ if .KubeadmVersion }}={{ .KubeadmVersion }}{{ end }}{{ end }} \
	docker.io || true

{{ if .IsPreReleaseVersion }}
curl -Lo kubeadm https://dl.k8s.io/release/{{ .KubeadmVersion }}/bin/linux/amd64/kubeadm \
    && chmod +x kubeadm \
	&& mv kubeadm /usr/bin/
{{ end }}

systemctl enable docker
systemctl start docker

{{ if .CloudConfigPath }}
cat > {{ .CloudConfigPath }} <<EOF
{{ .CloudConfig }}
EOF
{{ end }}

{{ if .ExternalProvider }}
cat > /etc/systemd/system/kubelet.service.d/20-pharmer.conf <<EOF
[Service]
Environment="KUBELET_EXTRA_ARGS=--node-labels=cloud.appscode.com/pool={{ .NodeGroupName }},node-role.kubernetes.io/node= --cloud-provider=external"
EOF
{{ else }}
cat > /etc/systemd/system/kubelet.service.d/20-pharmer.conf <<EOF
[Service]
Environment="KUBELET_EXTRA_ARGS=--node-labels=cloud.appscode.com/pool={{ .NodeGroupName }},node-role.kubernetes.io/node= {{ if  .CloudConfigPath }} --cloud-provider={{ .Provider }} --cloud-config={{ .CloudConfigPath }} {{ end }}"
EOF
{{end}}

systemctl daemon-reload
systemctl restart kubelet

kubeadm reset
{{ if .ConfigurationBucket }}
 {{ .ConfigurationBucket  }}
 source /etc/kubernetes/config.sh
 kubeadm join --token=${KUBEADM_TOKEN} {{ .APIServerAddress }}
{{ else }}
 kubeadm join --token={{ .KubeadmToken }} {{ .APIServerAddress }}
{{ end }}
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

	_ = template.Must(StartupScriptTemplate.New("ccm").Parse(`
until [ $(kubectl get pods -n kube-system -l k8s-app=kube-dns -o jsonpath={.items[0].status.phase} --kubeconfig /etc/kubernetes/admin.conf) == "Running" ]
do
   echo '.'
   sleep 5
done

kubectl apply -f "https://raw.githubusercontent.com/appscode/pharmer/master/cloud/providers/{{ .Provider }}/cloud-control-manager.yaml" --kubeconfig /etc/kubernetes/admin.conf

until [ $(kubectl get pods -n kube-system -l app=cloud-controller-manager -o jsonpath={.items[0].status.phase} --kubeconfig /etc/kubernetes/admin.conf) == "Running" ]
do
   echo '.'
   sleep 5
done

cat > /etc/systemd/system/kubelet.service.d/20-pharmer.conf <<EOF
[Service]
Environment="KUBELET_EXTRA_ARGS=--node-labels=cloud.appscode.com/pool={{ .NodeGroupName }} --cloud-provider=external"
EOF

NODE_NAME=$(uname -n)
kubectl taint nodes ${NODE_NAME} node.cloudprovider.kubernetes.io/uninitialized=true:NoSchedule --kubeconfig /etc/kubernetes/admin.conf

systemctl daemon-reload
systemctl restart kubelet

sleep 10
reboot
`))

	_ = template.Must(StartupScriptTemplate.New("calico").Parse(`
kubectl apply \
  -f http://docs.projectcalico.org/v2.3/getting-started/kubernetes/installation/hosted/kubeadm/1.6/calico.yaml \
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
