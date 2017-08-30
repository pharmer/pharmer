package cloud

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"text/template"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/go/errors"
	"github.com/appscode/go/net/httpclient"
	"github.com/appscode/pharmer/api"
	"github.com/golang/protobuf/jsonpb"
)

type TemplateData struct {
}

func GetTemplateData(ctx context.Context, cluster *api.Cluster) TemplateData {
	return TemplateData{}
}

func RenderStartupScript(ctx context.Context, cluster *api.Cluster, role string) (string, error) {
	var buf bytes.Buffer
	if err := StartupScriptTemplate.ExecuteTemplate(&buf, role, GetTemplateData(ctx, cluster)); err != nil {
		return "", err
	}
	return buf.String(), nil
}

var (
	StartupScriptTemplate = template.Must(template.New(api.RoleKubernetesMaster).Parse(`#!/usr/bin/env bash
set -e
set -x
cd ~

# logging startup script
LOGFILE=startup-script.log
exec > >(tee -a $LOGFILE)
exec 2>&1

# kill apt processes (E: Unable to lock directory /var/lib/apt/lists/)
kill $(ps aux | grep '[a]pt' | awk '{print $2}') || true

{{ template "prepare-host" . }}

# E: The method driver /usr/lib/apt/methods/https could not be found.
apt-get update -y
apt-get install -y apt-transport-https

curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
touch /etc/apt/sources.list.d/kubernetes.list
sh -c 'echo "deb http://apt.kubernetes.io/ kubernetes-xenial main" > /etc/apt/sources.list.d/kubernetes.list'

apt-get update -y
apt-get install -y \
    socat \
    ebtables \
    apt-transport-https \
    kubelet \
    kubeadm=1.7.3-01 \
    cloud-utils \
    jq

# ignore docker error (No sockets found via socket activation: make sure the service was started by systemd)
apt-get install -y docker.io || true

systemctl enable docker
systemctl start docker

kubeadm reset

mkdir -p /etc/kubernetes/pki
{{ template "setup-certs" . }}
chmod 600 /etc/kubernetes/pki/ca.key /etc/kubernetes/pki/front-proxy-ca.key

# get public and private ip addresses (comma separated)
PUBLIC_IP=$(curl ipinfo.io/ip)
PRIVATE_IP=$(hostname -I | tr ' ' '\n' | grep -E '^(192\.168|10\.|172\.1[6789]\.|172\.2[0-9]\.|172\.3[01]\.)' | xargs | tr ' ' ',')
# use public-ip if no private-ip found
if [ -z $PRIVATE_IP ]; then
	PRIVATE_IP=$PUBLIC_IP
fi

kubeadm init --apiserver-bind-port 6443 --token %v --apiserver-advertise-address ${PUBLIC_IP} --apiserver-cert-extra-sans=${PUBLIC_IP},${PRIVATE_IP},%v


mkdir -p ~/.kube
cp /etc/kubernetes/admin.conf ~/.kube/config`))

	_ = template.Must(StartupScriptTemplate.New(api.RoleKubernetesPool).Parse(`#!/usr/bin/env bash
set -e
set -x
cd ~

# logging startup script
LOGFILE=startup-script.log
exec > >(tee -a $LOGFILE)
exec 2>&1

# kill apt processes (E: Unable to lock directory /var/lib/apt/lists/)
kill $(ps aux | grep '[a]pt' | awk '{print $2}') || true

{{ template "prepare-host" . }}

# E: The method driver /usr/lib/apt/methods/https could not be found.
apt-get update -y
apt-get install -y apt-transport-https

curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
touch /etc/apt/sources.list.d/kubernetes.list
sh -c 'echo "deb http://apt.kubernetes.io/ kubernetes-xenial main" > /etc/apt/sources.list.d/kubernetes.list'

apt-get update -y
apt-get install -y \
	socat \
	ebtables \
	kubelet \
	kubeadm=1.7.3-01

# ignore docker error (No sockets found via socket activation: make sure the service was started by systemd)
apt-get install -y docker.io || true

systemctl enable docker
systemctl start docker

kubeadm reset
kubeadm join --token %v %v:6443
`))

	_ = template.Must(StartupScriptTemplate.New("prepare-host").Parse(``))

	_ = template.Must(StartupScriptTemplate.New("setup-certs").Parse(``))

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

func SaveInstancesInFirebase(opt *api.Cluster, ins *api.ClusterInstances) error {
	// TODO: FixIt
	// ins.Logger().Infof("Server is configured to skip startup config api")
	// store instances
	for _, v := range ins.Instances {
		if v.Status.ExternalIP != "" {
			fbPath, err := firebaseInstancePath(opt, v.Status.ExternalIP)
			if err != nil {
				return err // ors.FromErr(err).WithContext(ins).Err()
			}
			fmt.Println(fbPath)

			r2 := &proto.ClusterInstanceByIPResponse{
				Instance: &proto.ClusterInstance{
					Uid:        v.UID,
					ExternalId: v.Status.ExternalID,
					Name:       v.Name,
					ExternalIp: v.Status.ExternalIP,
					InternalIp: v.Status.InternalIP,
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
		cluster.Spec.StartupConfigToken), nil
}
