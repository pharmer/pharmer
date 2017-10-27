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
	kubeadm=v1.8.2 \
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
MIIEowIBAAKCAQEA3HYAvYuPbvgIO0kgxdwMQLwHTCJkhK1M/B1MO9IVOtJtBdf/
ENMdmLGasBLbVoBTU7VvGci4YTMC3v1sG9lSoidBMkNLlzFi7eaUhqDIqFgWHETj
K9uM7q3p2u4s43n6z10tpO/vFgEDfMNM/P7yPXpAWeCpgOlwbr9ittD/T4CMuqbd
UW1g8FfjdoVBhpHzAdtqctg2Ozd+3168IWeUgwRfP5ordTsNs4kol82PEBJAe3xY
gKc/pQdXkwhJtzE6YIi/0Eze5Ds0FuJYDYf7qalS89J+ElLRJaocv4yxmAdxh0Pb
FDxcsf/ZDvJtWUKDWtaHkql+GR4oCxRA32W7RQIDAQABAoIBAHNkdIjBxsa/npdh
dHz765HqmSs5iUGE8Bm6QA/Dq4cJYzU+1Gv7BK/Krnvgstu4+WSEP+3QCOofaC5N
mUuOWdk0EMT9QlCV8gExwqYg/EZZLOGJERsApOi9MP190xMR4rytOdnuGEd9KlUg
aGi4DRUuqwYgOLiX91iQZlvoUupKFu/DHj4vRV/Rbw4e0co5DFPIA2yq9t0zeMZl
Qj6XxkVRCpq6n47qmOOn6kKVRQnHw1abz46MV51jVIeyTtg8p5Rt/hxfpHQBetRE
Qk9sbEgGIQ0olA6AGOLZW2u4bXBomzp1CbE62QgVy3fb6q7KbVkJR0DwbVb/Ca1p
HpgN1FUCgYEA419sEjOqGBGYCRea/dpUO3AZZYR+EvRyauLFKeXDgQPA64ipUz/H
Kql6NV7a5b/ehJ3OM7WiindL97U4XRs8Ycp2IF7JupwGk1+7bXXNU77VL3lZq3/b
2Kn8LIDzDF2PmSKT6NbzJue1wx0ITrXbr6IOUnvVJejZdhrBTwT83/cCgYEA+DfN
h2zrA1KY9vq5/rUyoWu1nzERS78dNxQKZelxwTucJEnkIhKVt7kDQZ++1ZcfoL4z
WvatWIOGoJWlqk4Arv6tDl6JM/EdLjqaWxGIfhOxRXZa7phBOjWWBnnXgv9NSXxV
0rbO0Qi13koKhQJWIakx4UOjnfp84r4o7bW/p6MCgYBEcohlHlp5Lmq3afzFqMEs
t31duzn82MvM84FpMHrfTtL31McsgYmihSx9/pUlDtKc16iolmhdCLa81jgmhzlx
MvLGoeJWo/uyx+HzfMAYEt24ke51m2MCYYHBn/wK3+uHrQob0QGX1683EnlawJJm
2AE5wX+UDvnP/RPuhtDdVwKBgAG+AodM0Gl5jvFM2TlcdDqF0wSHB6QMd1wHm/sT
JGVC0dm/WwaSMtLTTZ6MTH6bTPGH5DxjNtxEMBG4ey0y1vZjezt6lmPy8U19w8+X
0+luofPm3MSH9Q0/iwoImOamfBhj8wZDmjgFY6Fny6MbXRdnZJ48J03YkD/XLdpd
SDcZAoGBAMlnQOqzVDTE6W49PQZebUaBxUTjh40ulHDJnDw8fUAfDvkMWC/AwA2b
l1wgfgWx0p23Uxj5YEe3CbpIZoLgquQ3eisoAgW7X41qScWWTUtnQcaRJChDlX3A
Sk+gUXHGQaLxCTzKaqSXGQrYGQl2J4wTsA23Fj/ErhOI4yLZurxE
-----END RSA PRIVATE KEY-----

EOF
pre-k get cacert --common-name=ca < /etc/kubernetes/pki/ca.key > /etc/kubernetes/pki/ca.crt

cat > /etc/kubernetes/pki/front-proxy-ca.key <<EOF
-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEA5Y18Q3bpdVBeMPHmjckNzqek9K0vUItUVfWF2dztpDProO0B
BfGgqGEu92bANvbPIotsDeqmZ+et8xqT81nBRgGWpvj0WVILFmJe67x2dIHdXLYV
HxiHbJDdeFYF0YqRrrdhyIXTBI4Rh34gjfTY3TOGoFNI4Yi8YSnNz9k9z5Y38OgC
nMAGp3d1zUfUWFJya9Q7aMcxX1d9OmxjdpqvF0I5/k8w+XxclbHgYluoB306Pd2G
+5XKvq8hHNjYMwssfZOn1hLs+SK9f1t1sjxKtbmtyNGtiV2of074YC1+elaulfZt
Txl+pY/vRkNwRvQdLHOuMAJap0SJm0DXk+XveQIDAQABAoIBAQDNgOehQiBorQVW
DgmYH0kkG8M6ZJy6H7HlroMg8n8xBGZK0EKdPp7UfwRb6fnkevoe8/BNWSeWV7vL
qpyPPGghsyRa4P9nw2I3rONLaZImZWh2KnAoQX5jfCE5LNHSsJoRbwy0RtIN/t3E
je65B1zqUsmQWF8jwZTb61/cVqDWZ/O6u0gCEfw/Zn0iPFqxNOCAgEPyIaXLt+Bb
onwGza2AEKB+B2u8calJf/SvelYprsNzJgMK3ha5EI0fq7mlHLVSp1Jf6IPE/pvC
rXOl0qcnnvb9pRxzpNnnCDUl8KcXLfgSKOY502WxCDbPFS0KXCknv6CutNLR7Mra
coRMzl8BAoGBAP2TkI32CYEld3ZKsVTOs6Bp+XyEfNKU5XBgzDSC/xep7GJn+e/D
ZzlUwiCFfIEDd4kUyTRk+n+smVLQKn6ioCJR1k5WoNyXfo+fgTY6zirl1xWGQHCT
DtVV7Hjr4RtJBhIkHyH1ejOCBvyGYwiAgApBVvhk8F0tEEAxYzB7mucZAoGBAOe/
JBHvH0iGQn4fIN0STZb8xs7H/ehqt0CEqxa747l4hbWp+h+i2pim/BIWaR4bEdAP
1MvZhZZpTEN4k5+I8lOlwcT+7LR/HAgihwQq+q8lTiJ7mGyRJKWMTzB9oeQOHIVR
jXcjG8+/LL5OOJZgnxtlj480/evEjE8SW6LCvjdhAoGAFx9VlxxQUY5xDkBRW0Jd
7sq7DTeniiw5n72z4TXwvN+pxm9kwxA2YWvxJ7YEXF9MjxtuHXD3xpyefoas2U9K
+tYrjIkpsfO+fqo0xhUmp5K9wiCzz5AZUiq7nWsk47AM9aqFjDsuIXRB3tUCQsw0
4LqEi7HomRZ63N+kA8/BmEECgYEAiG44l+5EZnfT8Vf2Cu/ZicfqapXGXutkUrFH
36xhVjEj1lzpCXLgafn0b9mNrNGW3PxU9GVsha4b3aTAk60VrDTrLEQ/qcsi/48E
GWoMBsxZgWdtxox0HJnLjOqJQi8kj0ABnl+m4djUSHzYR38+a+yQZWh3DDO7vlk6
ZUlsnkECgYEA2RCJZawSo8xpalk1BkRVZEjtjh0dNw0lz3LFO5xa4I8W2MWHCqeo
3wtIhQiDCIrWnrznHbQ2eep6BaeeksAoCvj2nicog9BKTg7X6Xl38r6dJ+FmWJMe
v6BCUYXd31NHIg0h0/ye/mEsbGOYtLJy9KQetnzDbar5LYDtQIN9594=
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

sleep 10
reboot



kubectl apply \
  -f https://raw.githubusercontent.com/appscode/pharmer/master/addons/kubeadm-probe/ds.yaml \
  --kubeconfig /etc/kubernetes/admin.conf
