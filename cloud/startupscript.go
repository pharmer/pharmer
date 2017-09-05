package cloud

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"text/template"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/go/net/httpclient"
	"github.com/appscode/pharmer/api"
	"github.com/golang/protobuf/jsonpb"
	"k8s.io/client-go/util/cert"
)

type TemplateData struct {
	KubernetesVersion string
	KubeadmVersion    string
	KubeadmToken      string
	CAKey             string
	FrontProxyKey     string
	APIServerHost     string
	ExtraDomains      string

	NetworkProvider string
}

func GetTemplateData(ctx context.Context, cluster *api.Cluster) TemplateData {
	return TemplateData{
		KubernetesVersion: cluster.Spec.KubernetesVersion,
		KubeadmVersion:    cluster.Spec.KubeadmVersion,
		KubeadmToken:      cluster.Spec.KubeadmToken,
		CAKey:             string(cert.EncodePrivateKeyPEM(CAKey(ctx))),
		FrontProxyKey:     string(cert.EncodePrivateKeyPEM(FrontProxyCAKey(ctx))),
		APIServerHost:     "",
		ExtraDomains:      cluster.Spec.ClusterExternalDomain,
		NetworkProvider:   cluster.Spec.NetworkProvider,
	}
}

func RenderStartupScript(ctx context.Context, cluster *api.Cluster, role string) (string, error) {
	var buf bytes.Buffer
	if err := StartupScriptTemplate.ExecuteTemplate(&buf, role, GetTemplateData(ctx, cluster)); err != nil {
		return "", err
	}
	return buf.String(), nil
}

var (
	StartupScriptTemplate = template.Must(template.New(api.RoleKubernetesMaster).Parse(`#!/bin/bash
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
	kubeadm{{ if .KubeadmVersion }}={{ .KubeadmVersion }}{{ end }} \
	cloud-utils \
	docker.io || true

curl -Lo cloudid https://cdn.appscode.com/binaries/cloudid/0.1.0-alpha.1/cloudid-linux-amd64 \
	&& chmod +x cloudid \
	&& mv cloudid /usr/bin/

systemctl enable docker
systemctl start docker

kubeadm reset

{{ template "setup-certs" . }}

kubeadm init \
	--apiserver-bind-port=6443 \
	--token={{ .KubeadmToken }} \
	--apiserver-advertise-address=$(cloudid get public-ips --all=false) \
	--apiserver-cert-extra-sans=$(cloudid get public-ips) \
	--apiserver-cert-extra-sans=$(cloudid get private-ips) \
	--apiserver-cert-extra-sans={{ .ExtraDomains }}

{{ if eq .NetworkProvider "flannel" }}
{{ template "flannel" . }}
{{ else if eq .NetworkProvider "calico" }}
{{ template "calico" . }}
{{ end }}
`))

	_ = template.Must(StartupScriptTemplate.New(api.RoleKubernetesPool).Parse(`#!/bin/bash
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
	kubeadmkubeadm{{ if .KubeadmVersion }}={{ .KubeadmVersion }}{{ end }} \
	docker.io || true

systemctl enable docker
systemctl start docker

kubeadm reset
kubeadm join --token={{ .KubeadmToken }} {{ .APIServerHost }}:6443
`))

	_ = template.Must(StartupScriptTemplate.New("prepare-host").Parse(``))

	_ = template.Must(StartupScriptTemplate.New("setup-certs").Parse(`
mkdir -p /etc/kubernetes/pki

cat > /etc/kubernetes/pki/ca.key <<EOF
{{ .CAKey }}
EOF
cloudid get cacert --common-name=ca < /etc/kubernetes/pki/ca.key > /etc/kubernetes/pki/ca.crt

cat > /etc/kubernetes/pki/front-proxy-ca.key <<EOF
{{ .FrontProxyKey }}
EOF
cloudid get cacert --common-name=front-proxy-ca < /etc/kubernetes/pki/front-proxy-ca.key > /etc/kubernetes/pki/front-proxy-ca.crt

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

func SaveInstancesInFirebase(opt *api.Cluster, instances []*api.Instance) error {
	// TODO: FixIt
	// ins.Logger().Infof("Server is configured to skip startup config api")
	// store instances
	for _, v := range instances {
		if v.Status.PublicIP != "" {
			fbPath, err := firebaseInstancePath(opt, v.Status.PublicIP)
			if err != nil {
				return err // ors.FromErr(err).WithContext(ins).Err()
			}
			fmt.Println(fbPath)

			r2 := &proto.ClusterInstanceByIPResponse{
				Instance: &proto.ClusterInstance{
					Uid:        v.UID,
					ExternalId: v.Status.ExternalID,
					Name:       v.Name,
					ExternalIp: v.Status.PublicIP,
					InternalIp: v.Status.PrivateIP,
					Sku:        v.Spec.SKU,
				},
			}

			var buf bytes.Buffer
			m := jsonpb.Marshaler{}
			err = m.Marshal(&buf, r2)
			if err != nil {
				return err // ors.FromErr(err).WithContext(ins).Err()
			}

			_, err = httpclient.New(nil, nil, nil).
				WithBaseURL(firebaseEndpoint).
				Call(http.MethodPut, fbPath, &buf, nil, false)
			if err != nil {
				return err // ors.FromErr(err).WithContext(ins).Err()
			}
		}
	}
	return nil
}

const firebaseEndpoint = "https://tigerworks-kube.firebaseio.com"

func firebaseInstancePath(cluster *api.Cluster, externalIP string) (string, error) {
	//l, err := api.FirebaseUid()
	//if err != nil {
	//	return "", errors.FromErr(err).Err()
	//}
	// https://www.firebase.com/docs/rest/guide/retrieving-data.html#section-rest-uri-params
	return fmt.Sprintf(`/k8s/%v/%v/%v/instance-by-ip/%v.json?auth=%v`,
		"l",
		"",           /* cluster.Namespace */
		cluster.Name, // phid is grpc api
		strings.Replace(externalIP, ".", "_", -1),
		cluster.Spec.StartupConfigToken), nil
}
