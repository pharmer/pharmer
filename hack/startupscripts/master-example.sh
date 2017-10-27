#!/bin/bash
set -x
set -o errexit
set -o nounset
set -o pipefail

# log to /var/log/startup-script.log
exec > >(tee -a /var/log/startup-script.log)
exec 2>&1

# kill apt processes (E: Unable to lock directory /var/lib/apt/lists/)
kill $(ps aux | grep '[a]pt' | awk '{print $2}') || true



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
	kubectl \
	kubelet \
	kubeadm \
	cloud-utils \
	docker.io || true



curl -Lo pre-k https://cdn.appscode.com/binaries/pre-k/0.1.0-alpha.5/pre-k-linux-amd64 \
	&& chmod +x pre-k \
	&& mv pre-k /usr/bin/

systemctl enable docker
systemctl start docker

cat > /etc/systemd/system/kubelet.service.d/20-pharmer.conf <<EOF
[Service]
Environment="KUBELET_EXTRA_ARGS=--node-labels=cloud.appscode.com/pool=master --cloud-provider= "
EOF

systemctl daemon-reload
systemctl restart kubelet

kubeadm reset


mkdir -p /etc/kubernetes/pki

cat > /etc/kubernetes/pki/ca.key <<EOF
-----BEGIN RSA PRIVATE KEY-----
MIIEogIBAAKCAQEAz843QQhXpjqBKXcJxw90Qoqsr5Z/jsqcut8EVUq83vV1JHGK
sVSf8ohmJhKs5Qh6zBqaGZ8ZJWrfKNrWFdlnrCYC7fM3XL+5ApOI/wqHIRyZT/eZ
kEfUUZHg5PpRCd6bhwuq6it5CQozYiXNT4K+aFKojk2MeOZdhS9yD5tH/g96ANQP
39bFQ3QQZpL60BTB+ZVMTLuLGVvZDz4JTXtk/LuWJJArCMrjL8tconXI9wLuipjt
IsY1qvzz+CoiOXMDzsh3MOR0BrT9EpjucsBd99+ddgMN0xDgEYpY4vrH98dYCcvE
giock/lUiaEcr3hX/oYkDT1l5kD7dKZWkpfMMwIDAQABAoIBAHJJHHRErUe7h0uR
ryvuIOdzsvNCltamMbpIau6pkuQgJJOtajSKsQjG4T7xKGsx1a8otjV/HWpJs3+Z
kwIjNfQkV5ocGAeHXa3ADCkP1i9sthiXuLnz9x4BV6k2zZja97g2v4HX9NH27Tl+
RsMCyctAInlYxve64hYceOOCZ/6d7+mAQF8W7eSzJGWXHMk5K7d6Jwcx7jBS/WS2
Czsv9pcGu8ckOPuU96Xi49v1DHjH0i+vS0oWNnHtjWeY4LTuAVuinxEDOlL/fpnz
3E65yFdgs4fqimd3LZZDwbDhGpZND/3gXKHMKTOBVKxIxweMQ8I0APi2Ctbq9PJg
YZ4+nxECgYEA1Wx35GJMRo/Ph/AO6BECkCStOItj89uZvXd61dQKL/N9K2AlBeLl
1evMPiKsDjN9FwNm2CBRiTLdqtrIhgzPiTf3i7hx4I2yWLUUN5uyQ621eg06tYt2
uZcn3M5ssIiUWk9DjqHIOEQgeHo1kyClcADPTBQu7dzWd5vuIPAcU7kCgYEA+ULT
68x4aWkiyea/fbsZ29tbTn3TKqsuF04R09tvj5oOluAeaE4EkIXwzVMHJJZG6bFq
1t+DrYMnEVY26fwqSrLbf2u7VT62lwfOx1Bhq/ak/jyjpoA6LeIJuwa/viFSRN52
NXL/DTuF0H4naB3OBHZRi3OBdrAESbHqY72R7UsCgYB3gHzBTKkY+X1iyHAQUTX2
MBMuDh6xdMzo4fXNtSTfJJ95oiQY36uB1L5QLGnaqcnpEOaNLct53xlviYGuTY4H
b2cUvPpGmhC6yum/GVb/vkxXQwEUljqsQI75fDwvvMoUpz1UqBHML5le3E8TSrxX
spxgJQ0B8x3Da3QyzT+PQQKBgED0LzFFKSOe6Bfg74meFhD6yoJbu4lk7i/YgkDI
7/tl0+NxJ7taiUn3/VYkCrp4BqajOwofWLsAcE/OPaUftw2cKiK8OibunrogqLu7
sJgVP82Yk7SxuXd3bb209oZfPIcByaAIBXq3Rhmcpjw1eBgllP5X7Sa2m4dwu3me
TsadAoGAbYVJwKjMl9KSEqr0ZXz98TcvRdQ1oNNA6E6Q60XGwjfnQ8/HczsTpAZ0
02PyxG/tJQ1rG8/J3/c40XQNWdBNM1AWxWbU3QH3NDMfVyhLGtMt4PKCLQtQegIT
U9IkOE9BRll0z/jaTVhXn6mlC6gcwzxRpJPB7iMYLHs/7+CoupY=
-----END RSA PRIVATE KEY-----

EOF
pre-k get cacert --common-name=ca < /etc/kubernetes/pki/ca.key > /etc/kubernetes/pki/ca.crt

cat > /etc/kubernetes/pki/front-proxy-ca.key <<EOF
-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEA+4XhmkN7pUPDzUpoMvKzyV30jWGBEzUkiftS72213uECUSOL
xqmgaewYB8+7MBiq/q02sSBjaZs0yU2Z1hHoOvxeVhbpmXl0dTNVAnOmVl59Rl+5
VSD0oXbn0JpwlGf3GlA7kkLAgky+lCuD1U5Uq/nqKwLwgMPkbLOdE/mXSOXLOMaA
FCxXRBq0l/M/3hvW19JZZrTgX7AXVEUVrefTXly9QfBM+P5R5t0jTR5PmG9f9P9K
aOiFHPQVGx49pyPadg/QflcB3EZk2aGB7FpS3IFg03dLrDRcnmd5+zMMcxFwETEI
uu57rhcE6xdwdkMGBcu2SiUOdiQgMMnd5RqkdQIDAQABAoIBAQDRiV5BynhGXKbQ
7mzSDNj0J36lDZafLsWK4cHczvQVgjQQ7mDylruZomL+lvMlhVdmpVyLwSSwhOk7
zpca/H4QLdBVPe9LuR/ox2PJkBkBmOQabYKTRcomfU1vvkmNiPMVi8Ok/FEt+8tE
2t+QIxpszt1jCabcTtWMLTHtwx9iTM7t1k/ijFiFoGlUkFalxlL68O0XyBxLBsF3
aFlpFx3hx8GeTGLKP1LkdC5V9nChorTt0jPT72552SsvGzAgjJXwgOYaZVT18Co3
6jvJQtqE4SfiyTT2/atRWDWYW3aIpYbv3LKI+htiZE2pF2FvDvb8bEWdC5WjjBgf
4RMO9IlhAoGBAP0Ro2w9YESoaSPWML5mO4xp6cF4q0j/4eg93cTpl1NMP5x6YqgA
Cq5iJIxz8skY6LkfDVcvKZB9poRlk0TFGDQZQzEkVRALGhaKW0ozoTZb4l6/1JOo
UI7Efnc86DUkTFA40xN+ZRUMzSJlFSGMZBagCeN4Gv5BDXaWgDudasAtAoGBAP5v
qL2bKaFJc1k+vUWdKGEK6+1YIKtzlAoLXxeONOzK9h1QbB5MrlAK/y6BaKzd+RO4
DE2qGtjDol+QQBIRfToSZrhr+TK4N9g0iTppIqZyTYCElg4lC/tbYLDHVkODUqzI
Sl2o7CD0jjwX/jxfH9WDT2BqGXfEDyEFD/hXClppAoGBAJ9zlqGjhl2cMyr2UR4W
chzFuxhA/MNxHpjjKjFg8Qz19PqWDkVGSXSYt+ibQfU9MZ06mIA/YhP9Z4byLTle
aiKhbDmxZ7tFE3nyURTOtRv4BwimG72xhph97CUbBkA9j6vYMf6NXAlIWJP8VrUd
rQ6oZbKiQXfD60S2DTQCunk9AoGADnsobXIxUl6t0/yAJUAmli9a8i073sY7PL8c
8GhFltyOWWjEXo0atq+JiooO/Re9H2QwPxNZZ9DqounA28ZnDDET65cpnbtiYknL
LaniMPr8cj4ZlECDiBDRVf5iaIFG6VKU+POuTMnedokfDSyU0UAh+mjPfkOIYOa6
2/WIP1ECgYEAs8UliLRvb5f7ZG+lbaJN0/1wBFXra0a+YhumMiFARVDOG7qx+j8v
D3Z11DCCyEiO6JOMSCMlJxPIYLwaKo1+gNjqByiEX5u/yPFgMvnyLsNeAo9G2bQB
ycYmrMNooJCnxzsNPpTduh87sJPDJydiES95PSiBf5b16ckqokkrfCc=
-----END RSA PRIVATE KEY-----

EOF
pre-k get cacert --common-name=front-proxy-ca < /etc/kubernetes/pki/front-proxy-ca.key > /etc/kubernetes/pki/front-proxy-ca.crt

chmod 600 /etc/kubernetes/pki/ca.key /etc/kubernetes/pki/front-proxy-ca.key




mkdir -p /etc/kubernetes/kubeadm


cat > /etc/kubernetes/kubeadm/config.yaml <<EOF
api:
  advertiseAddress: ""
  bindPort: 6443
apiVersion: kubeadm.k8s.io/v1alpha1
certificatesDir: ""
cloudProvider: external
etcd:
  caFile: ""
  certFile: ""
  dataDir: ""
  endpoints: null
  image: ""
  keyFile: ""
imageRepository: ""
kind: MasterConfiguration
kubernetesVersion: 1.8.0
networking:
  dnsDomain: ""
  podSubnet: ""
  serviceSubnet: ""
nodeName: ""
token: ""
tokenTTL: 0s
unifiedControlPlaneImage: ""

EOF


pre-k merge master-config \
	--config=/etc/kubernetes/kubeadm/config.yaml \
	--apiserver-bind-port=6443 \
	--apiserver-advertise-address=$(pre-k get public-ips --all=false) \
	--apiserver-cert-extra-sans=$(pre-k get public-ips --routable) \
	--apiserver-cert-extra-sans=$(pre-k get private-ips) \
	--apiserver-cert-extra-sans= \
	--kubernetes-version=1.8.0 \
	> /etc/kubernetes/kubeadm/config.yaml
kubeadm init --config=/etc/kubernetes/kubeadm/config.yaml --skip-token-print



kubectl apply \
  -f http://docs.projectcalico.org/v2.3/getting-started/kubernetes/installation/hosted/kubeadm/1.6/calico.yaml \
  --kubeconfig /etc/kubernetes/admin.conf



kubectl apply \
  -f https://raw.githubusercontent.com/appscode/pharmer/master/addons/kubeadm-probe/ds.yaml \
  --kubeconfig /etc/kubernetes/admin.conf

mkdir -p ~/.kube
sudo cp -i /etc/kubernetes/admin.conf ~/.kube/config
sudo chown $(id -u):$(id -g) ~/.kube/config



until [ $(kubectl get pods -n kube-system -l k8s-app=kube-dns -o jsonpath='{.items[0].status.phase}' --kubeconfig /etc/kubernetes/admin.conf) == "Running" ]
do
   echo '.'
   sleep 5
done

kubectl apply -f "https://raw.githubusercontent.com/appscode/pharmer/master/cloud/providers/digitalocean/cloud-control-manager.yaml" --kubeconfig /etc/kubernetes/admin.conf

until [ $(kubectl get pods -n kube-system -l app=cloud-controller-manager -o jsonpath='{.items[0].status.phase}' --kubeconfig /etc/kubernetes/admin.conf) == "Running" ]
do
   echo '.'
   sleep 5
done

cat > /etc/systemd/system/kubelet.service.d/20-pharmer.conf <<EOF
[Service]
Environment="KUBELET_EXTRA_ARGS=--node-labels=cloud.appscode.com/pool=master --cloud-provider=external "
EOF

NODE_NAME=$(uname -n)
kubectl taint nodes ${NODE_NAME} node.cloudprovider.kubernetes.io/uninitialized=true:NoSchedule --kubeconfig /etc/kubernetes/admin.conf

systemctl daemon-reload
systemctl restart kubelet

# sleep 10
# reboot
