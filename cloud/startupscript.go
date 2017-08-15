package cloud

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/go/net/httpclient"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/context"
	"github.com/golang/protobuf/jsonpb"
)

func RenderKubeadmStarter(cluster *api.Cluster, sku string) string {
	return fmt.Sprintf(`#!/bin/bash -e
	/usr/bin/apt-get install -y apt-transport-https
curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -
touch /etc/apt/sources.list.d/kubernetes.list
sh -c 'echo "deb http://apt.kubernetes.io/ kubernetes-xenial main" > /etc/apt/sources.list.d/kubernetes.list'

apt-get update -y
apt-get install -y \
    socat \
    ebtables \
    docker.io \
    apt-transport-https \
    kubelet \
    kubeadm=1.7.0-00 \
    cloud-utils


systemctl enable docker
systemctl start docker
PUBLICIP=$(curl ipinfo.io/ip)
PRIVATEIP=$(ifconfig | grep -A 1 ens4 | grep inet | cut -d ":" -f 2 | cut -d " " -f 1 | xargs)

kubeadm reset
kubeadm init --apiserver-bind-port 6443  --apiserver-advertise-address ${PUBLICIP} --apiserver-cert-extra-sans ${PUBLICIP} ${PRIVATEIP}
kubectl apply \
  -f http://docs.projectcalico.org/v2.3/getting-started/kubernetes/installation/hosted/kubeadm/1.6/calico.yaml \
  --kubeconfig /etc/kubernetes/admin.conf

mkdir -p $HOME/.kube
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config
`)
}

func RenderKubeadmMasterStarter(cluster *api.Cluster, cert string) string {
	return fmt.Sprintf(`#!/bin/bash -e
#set -o errexit
#set -o nounset
#set -o pipefail


apt-get install -y wget curl apt-transport-https

curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -
touch /etc/apt/sources.list.d/kubernetes.list
sh -c 'echo "deb http://apt.kubernetes.io/ kubernetes-xenial main" > /etc/apt/sources.list.d/kubernetes.list'

apt-get update -y
apt-get install -y \
    socat \
    ebtables \
    docker.io \
    kubelet \
    kubeadm=1.7.0-00 \
    cloud-utils


systemctl enable docker
systemctl start docker

mkdir -p /etc/kubernetes/pki
PUBLICIP=$(curl ipinfo.io/ip)
PRIVATEIP=$(ip route get 8.8.8.8 | awk '{print $NF; exit}')

kubeadm reset
%v

chmod 600 /etc/kubernetes/pki/ca.key /etc/kubernetes/pki/front-proxy-ca.key
kubeadm init --apiserver-bind-port 6443 --token %v  --apiserver-advertise-address ${PUBLICIP} --apiserver-cert-extra-sans ${PUBLICIP} ${PRIVATEIP} --pod-network-cidr 10.244.0.0/16 --kubernetes-version %v

kubectl apply \
  -f https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml \
  --kubeconfig /etc/kubernetes/admin.conf

kubectl apply \
  -f https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel-rbac.yml \
  --kubeconfig /etc/kubernetes/admin.conf

mkdir -p ~/.kube
sudo cp -i /etc/kubernetes/admin.conf ~/.kube/config`, cert, cluster.KubeadmToken, cluster.KubeVersion)
}

//   \
func RenderKubeadmNodeStarter(cluster *api.Cluster) string {
	return fmt.Sprintf(`#!/bin/bash -e
#set -o errexit
#set -o nounset
#set -o pipefail

apt-get install -y wget curl apt-transport-https
curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -
touch /etc/apt/sources.list.d/kubernetes.list
sh -c 'echo "deb http://apt.kubernetes.io/ kubernetes-xenial main" > /etc/apt/sources.list.d/kubernetes.list'

apt-get update -y
apt-get install -y \
    socat \
    ebtables \
    docker.io \
    kubelet \
    kubeadm=1.7.0-00

systemctl enable docker
systemctl start docker

kubeadm reset
kubeadm join --token %v %v:6443
`, cluster.KubeadmToken, cluster.MasterExternalIP)
}

// firebase certs upload @dipta

func UploadCertInFirebase(ctx context.Context, cluster *api.Cluster, certName string, certData string) error {
	cert, err := base64.StdEncoding.DecodeString(certData)
	if err != nil {
		return errors.FromErr(err).WithContext(ctx).Err()
	}

	resp := &proto.ClusterStartupConfigResponse{
		Configuration: string(cert),
	}
	m := jsonpb.Marshaler{}
	data, err := m.MarshalToString(resp)
	if err != nil {
		return errors.FromErr(err).WithContext(ctx).Err()
	}

	fbPath, err := FirebaseCertPath(ctx, cluster, certName)
	if err != nil {
		return errors.FromErr(err).WithContext(ctx).Err()
	}

	_, err = httpclient.New(nil, nil, nil).
		WithBaseURL(firebaseEndpoint).
		Call(http.MethodPut, fbPath, bytes.NewBufferString(data), nil, false)
	if err != nil {
		return errors.FromErr(err).WithContext(ctx).Err()
	}

	return nil
}

func UploadAllCertsInFirebase(ctx context.Context, cluster *api.Cluster) error {
	kubeadmCerts := [][]string{
		{"ca-crt", cluster.CaCert},
		{"ca-key", cluster.CaKey},
		{"front-proxy-ca-crt", cluster.FrontProxyCaCert},
		{"front-proxy-ca-key", cluster.FrontProxyCaKey}}

	for _, cert := range kubeadmCerts {
		err := UploadCertInFirebase(ctx, cluster, cert[0], cert[1])
		if err != nil {
			return errors.FromErr(err).WithContext(ctx).Err()
		}
	}

	return nil
}

func FirebaseCertPath(ctx context.Context, cluster *api.Cluster, certName string) (string, error) {
	l, err := api.FirebaseUid()
	if err != nil {
		return "", errors.FromErr(err).WithContext(ctx).Err()
	}
	// https://www.firebase.com/docs/rest/guide/retrieving-data.html#section-rest-uri-params
	return fmt.Sprintf(`/k8s/%v/%v/%v/kubernetes/context/%v/pki/%v.json?auth=%v`,
		l,
		"team",       // TODO: FixIt!
		cluster.Name, // phid is grpc api
		cluster.ContextVersion,
		certName,
		cluster.StartupConfigToken), nil
}

func FireBaseCertDownloadCmd(ctx context.Context, cluster *api.Cluster) (string, error) {
	kubeadmCerts := [][]string{
		{"ca-crt", "/etc/kubernetes/pki/ca.crt"},
		{"ca-key", "/etc/kubernetes/pki/ca.key"},
		{"front-proxy-ca-crt", "/etc/kubernetes/pki/front-proxy-ca.crt"},
		{"front-proxy-ca-key", "/etc/kubernetes/pki/front-proxy-ca.key"}}

	certCmd := ""

	for _, cert := range kubeadmCerts {
		path, err := FirebaseCertPath(ctx, cluster, cert[0])
		if err != nil {
			return "", errors.FromErr(err).WithContext(ctx).Err()
		}

		path = firebaseEndpoint + path
		cmd := fmt.Sprintf("curl -fsSL '%v' | jq -r '.configuration' > %v\n", path, cert[1])
		certCmd = certCmd + cmd
	}

	return certCmd, nil
}

// DO kubeadm startup script @dipta

func RenderDoKubeMaster(ctx context.Context, cluster *api.Cluster, cmd string) string {
	return fmt.Sprintf(`#!/usr/bin/env bash
set -e
cd ~

LOGFILE=startup-script.log
exec > >(tee -a $LOGFILE)
exec 2>&1

curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -
touch /etc/apt/sources.list.d/kubernetes.list
sh -c 'echo "deb http://apt.kubernetes.io/ kubernetes-xenial main" > /etc/apt/sources.list.d/kubernetes.list'

apt-get update -y
apt-get install -y \
    socat \
    ebtables \
    docker.io \
    apt-transport-https \
    kubelet \
    kubeadm=1.7.0-00 \
    cloud-utils \
    jq


systemctl enable docker
systemctl start docker

PUBLICIP=$(curl ipinfo.io/ip)
PRIVATEIP=$(ip route get 8.8.8.8 | awk '{print $NF; exit}')

kubeadm reset

mkdir -p /etc/kubernetes/pki
%v
chmod 600 /etc/kubernetes/pki/ca.key /etc/kubernetes/pki/front-proxy-ca.key

kubeadm init --apiserver-bind-port 6443 --token %v --apiserver-advertise-address ${PUBLICIP} --apiserver-cert-extra-sans=${PUBLICIP},${PRIVATEIP},%v

# Thanks Kelsey :)
kubectl apply \
  -f http://docs.projectcalico.org/v2.3/getting-started/kubernetes/installation/hosted/kubeadm/1.6/calico.yaml \
  --kubeconfig /etc/kubernetes/admin.conf

mkdir -p ~/.kube
cp /etc/kubernetes/admin.conf ~/.kube/config`,
		cmd,
		cluster.KubeadmToken,
		ctx.Extra().ExternalDomain(cluster.Name))
}

func RenderDoKubeNode(cluster *api.Cluster) string {
	return fmt.Sprintf(`#!/usr/bin/env bash
set -e
cd ~

LOGFILE=startup-script.log
exec > >(tee -a $LOGFILE)
exec 2>&1

sudo curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -
sudo touch /etc/apt/sources.list.d/kubernetes.list
sudo sh -c 'echo "deb http://apt.kubernetes.io/ kubernetes-xenial main" > /etc/apt/sources.list.d/kubernetes.list'

sudo apt-get update -y
sudo apt-get install -y \
    socat \
    ebtables \
    docker.io \
    apt-transport-https \
    kubelet \
    kubeadm=1.7.0-00

sudo systemctl enable docker
sudo systemctl start docker

sudo -E kubeadm reset
sudo -E kubeadm join --token %v %v:6443`, cluster.KubeadmToken, cluster.MasterExternalIP)
}

// -----------------------------------------------------------------------------------

// This is called from a /etc/rc.local script, so always use full path for any command
func RenderKubeStarter(cluster *api.Cluster, sku, cmd string) string {
	return fmt.Sprintf(`#!/bin/bash -e
set -o errexit
set -o nounset
set -o pipefail

export LC_ALL=en_US.UTF-8
export LANG=en_US.UTF-8
/usr/bin/apt-get update || true
/usr/bin/apt-get install -y wget curl aufs-tools

%v

/usr/bin/wget %v -O start-kubernetes
/bin/chmod a+x start-kubernetes
/bin/echo $CONFIG | ./start-kubernetes --v=3 --sku=%v
/bin/rm start-kubernetes
`,
		cmd, "FixIt!" /*cluster.KubeStarterURL*/, sku)
}

// http://askubuntu.com/questions/9853/how-can-i-make-rc-local-run-on-startup
func RenderKubeInstaller(cluster *api.Cluster, sku, role, cmd string) string {
	return fmt.Sprintf(`#!/bin/bash
cat >/etc/kube-installer.sh <<EOF
%v
rm /lib/systemd/system/kube-installer.service
systemctl daemon-reload
exit 0
EOF
chmod +x /etc/kube-installer.sh

cat >/lib/systemd/system/kube-installer.service <<EOF
[Unit]
Description=Install Kubernetes Master

[Service]
Type=simple
ExecStart=/bin/bash -e /etc/kube-installer.sh
Restart=on-failure
StartLimitInterval=5

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable kube-installer.service
`, strings.Replace(RenderKubeStarter(cluster, sku, cmd), "$", "\\$", -1))
}

func SaveInstancesInFirebase(opt *api.Cluster, ins *api.ClusterInstances) error {
	// TODO: FixIt
	// ins.Logger().Infof("Server is configured to skip startup config api")
	// store instances
	for _, v := range ins.Instances {
		if v.ExternalIP != "" {
			fbPath, err := firebaseInstancePath(opt, v.ExternalIP)
			if err != nil {
				return err // ors.FromErr(err).WithContext(ins).Err()
			}
			fmt.Println(fbPath)

			r2 := &proto.ClusterInstanceByIPResponse{
				Instance: &proto.ClusterInstance{
					Phid:       v.PHID,
					ExternalId: v.ExternalID,
					Name:       v.Name,
					ExternalIp: v.ExternalIP,
					InternalIp: v.InternalIP,
					Sku:        v.SKU,
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

func UploadStartupConfigInFirebase(ctx context.Context, cluster *api.Cluster) error {
	ctx.Logger().Infof("Server is configured to skip startup config api")
	{
		cfg, err := cluster.StartupConfigResponse(api.RoleKubernetesMaster)
		if err != nil {
			return errors.FromErr(err).WithContext(ctx).Err()
		}
		fbPath, err := firebaseStartupConfigPath(cluster, api.RoleKubernetesMaster)
		if err != nil {
			return errors.FromErr(err).WithContext(ctx).Err()
		}
		fmt.Println(fbPath)

		_, err = httpclient.New(nil, nil, nil).
			WithBaseURL(firebaseEndpoint).
			Call(http.MethodPut, fbPath, bytes.NewBufferString(cfg), nil, false)
		if err != nil {
			return errors.FromErr(err).WithContext(ctx).Err()
		}
	}
	{
		// store startup config
		cfg, err := cluster.StartupConfigResponse(api.RoleKubernetesPool)
		if err != nil {
			return errors.FromErr(err).WithContext(ctx).Err()
		}
		fbPath, err := firebaseStartupConfigPath(cluster, api.RoleKubernetesPool)
		if err != nil {
			return errors.FromErr(err).WithContext(ctx).Err()
		}
		fmt.Println(fbPath)

		_, err = httpclient.New(nil, nil, nil).
			WithBaseURL(firebaseEndpoint).
			Call(http.MethodPut, fbPath, bytes.NewBufferString(cfg), nil, false)
		if err != nil {
			return errors.FromErr(err).WithContext(ctx).Err()
		}
	}
	return nil
}

func StartupConfigFromFirebase(cluster *api.Cluster, role string) string {
	url, _ := firebaseStartupConfigPath(cluster, role)
	return fmt.Sprintf(`CONFIG=$(/usr/bin/wget -qO- '%v' 2> /dev/null)`, url)
}

func StartupConfigFromAPI(cluster *api.Cluster, role string) string {
	// TODO(tamal): Use wget instead of curl
	return fmt.Sprintf(`CONFIG=$(/usr/bin/wget -qO- '%v/kubernetes/v1beta1/clusters/%v/startup-script/%v/context-versions/%v/json' --header='Authorization: Bearer %v:%v' 2> /dev/null)`,
		"", // system.PublicAPIHttpEndpoint(),
		cluster.PHID,
		role,
		"", /* cluster.Namespace */
		cluster.ContextVersion,
		cluster.StartupConfigToken)
}

const firebaseEndpoint = "https://tigerworks-kube.firebaseio.com"

func firebaseStartupConfigPath(cluster *api.Cluster, role string) (string, error) {
	l, err := api.FirebaseUid()
	if err != nil {
		return "", errors.FromErr(err).WithContext(nil).Err()
	}
	// https://www.firebase.com/docs/rest/guide/retrieving-data.html#section-rest-uri-params
	return fmt.Sprintf(`/k8s/%v/%v/%v/startup-script/%v/context-versions/%v.json?auth=%v`,
		l,
		"",           /* cluster.Namespace */
		cluster.Name, // phid is grpc api
		role,
		cluster.ContextVersion,
		cluster.StartupConfigToken), nil
}

func firebaseInstancePath(cluster *api.Cluster, externalIP string) (string, error) {
	l, err := api.FirebaseUid()
	if err != nil {
		return "", errors.FromErr(err).Err()
	}
	// https://www.firebase.com/docs/rest/guide/retrieving-data.html#section-rest-uri-params
	return fmt.Sprintf(`/k8s/%v/%v/%v/instance-by-ip/%v.json?auth=%v`,
		l,
		"",           /* cluster.Namespace */
		cluster.Name, // phid is grpc api
		strings.Replace(externalIP, ".", "_", -1),
		cluster.StartupConfigToken), nil
}
