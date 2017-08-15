package cloud

import (
	"bytes"
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

func RenderKubeadmStarter(opt *api.ScriptOptions, sku string) string {
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

func RenderKubeadmMasterStarter(opt *api.ScriptOptions, cert string) string {
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
sudo cp -i /etc/kubernetes/admin.conf ~/.kube/config`, cert, opt.Ctx.KubeadmToken, opt.Ctx.KubeVersion)
}

//   \
func RenderKubeadmNodeStarter(opt *api.ScriptOptions) string {
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
	`, opt.Ctx.KubeadmToken, opt.Ctx.MasterExternalIP)
}

// This is called from a /etc/rc.local script, so always use full path for any command
func RenderKubeStarter(opt *api.ScriptOptions, sku, cmd string) string {
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
		cmd, opt.KubeStarterURL, sku)
}

// http://askubuntu.com/questions/9853/how-can-i-make-rc-local-run-on-startup
func RenderKubeInstaller(opt *api.ScriptOptions, sku, role, cmd string) string {
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
`, strings.Replace(RenderKubeStarter(opt, sku, cmd), "$", "\\$", -1))
}

func SaveInstancesInFirebase(opt *api.ScriptOptions, ins *api.ClusterInstances) error {
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
		fbPath, err := firebaseStartupConfigPath(cluster.NewScriptOptions(), api.RoleKubernetesMaster)
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
		fbPath, err := firebaseStartupConfigPath(cluster.NewScriptOptions(), api.RoleKubernetesPool)
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

func StartupConfigFromFirebase(opt *api.ScriptOptions, role string) string {
	url, _ := firebaseStartupConfigPath(opt, role)
	return fmt.Sprintf(`CONFIG=$(/usr/bin/wget -qO- '%v' 2> /dev/null)`, url)
}

func StartupConfigFromAPI(opt *api.ScriptOptions, role string) string {
	// TODO(tamal): Use wget instead of curl
	return fmt.Sprintf(`CONFIG=$(/usr/bin/wget -qO- '%v/kubernetes/v1beta1/clusters/%v/startup-script/%v/context-versions/%v/json' --header='Authorization: Bearer %v:%v' 2> /dev/null)`,
		"", // system.PublicAPIHttpEndpoint(),
		opt.PHID,
		role,
		opt.Namespace,
		opt.ContextVersion,
		opt.StartupConfigToken)
}

const firebaseEndpoint = "https://tigerworks-kube.firebaseio.com"

func firebaseStartupConfigPath(opt *api.ScriptOptions, role string) (string, error) {
	l, err := api.FirebaseUid()
	if err != nil {
		return "", errors.FromErr(err).WithContext(nil).Err()
	}
	// https://www.firebase.com/docs/rest/guide/retrieving-data.html#section-rest-uri-params
	return fmt.Sprintf(`/k8s/%v/%v/%v/startup-script/%v/context-versions/%v.json?auth=%v`,
		l,
		opt.Namespace,
		opt.Name, // phid is grpc api
		role,
		opt.ContextVersion,
		opt.StartupConfigToken), nil
}

func firebaseInstancePath(opt *api.ScriptOptions, externalIP string) (string, error) {
	l, err := api.FirebaseUid()
	if err != nil {
		return "", errors.FromErr(err).Err()
	}
	// https://www.firebase.com/docs/rest/guide/retrieving-data.html#section-rest-uri-params
	return fmt.Sprintf(`/k8s/%v/%v/%v/instance-by-ip/%v.json?auth=%v`,
		l,
		opt.Namespace,
		opt.Name, // phid is grpc api
		strings.Replace(externalIP, ".", "_", -1),
		opt.StartupConfigToken), nil
}
