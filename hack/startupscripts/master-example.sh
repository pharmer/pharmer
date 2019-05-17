#!/bin/bash
set -x
set -o errexit
set -o nounset
set -o pipefail

# log to /var/log/pharmer.log
exec > >(tee -a /var/log/pharmer.log)
exec 2>&1

# kill apt processes (E: Unable to lock directory /var/lib/apt/lists/)
kill $(ps aux | grep '[a]pt' | awk '{print $2}') || true

apt-get update -y
apt-get install -y apt-transport-https curl ca-certificates

curl -fSsL https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
echo 'deb http://apt.kubernetes.io/ kubernetes-xenial main' >/etc/apt/sources.list.d/kubernetes.list

add-apt-repository -y ppa:gluster/glusterfs-3.10

apt-get update -y
apt-get install -y cron docker.io ebtables git glusterfs-client haveged kubectl kubelet nfs-common socat kubeadm ntp || true

curl -Lo pre-k https://cdn.appscode.com/binaries/pre-k/0.1.0-alpha.6/pre-k-linux-amd64 &&
  chmod +x pre-k &&
  mv pre-k /usr/bin/

systemctl enable docker
systemctl start docker

cat >/etc/systemd/system/kubelet.service.d/20-pharmer.conf <<EOF
[Service]
Environment="KUBELET_EXTRA_ARGS=--node-labels=cluster.pharmer.io/pool=master --cloud-provider=external "
EOF

systemctl daemon-reload
systemctl restart kubelet

kubeadm reset

mkdir -p /etc/kubernetes/pki

cat >/etc/kubernetes/pki/ca.key <<EOF
-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEAxlWMEyTBU4qVOdu7z51AX4BFX092/epnqznUMBHIR+XQrnA2
gbJyR60GYgkhrEbpaR/g3Z0dw3X3QvA9IzfBx/sH+KNtPZEzN8PKOEF2Sa4vVSvV
qzLvUXVaezXXyrNzKcfmKAuHlNAR89cECNTpB6QbKYfHogXTw8iODMqjiNWRh60V
BNYP76wU3vRMjmKsLKRT4C1F6OMrE8R6gmS2PTQpsmBWJWGpt+Hy2tcGnrbupVJT
uuGpU9GKl46TsRUR0+6EuhKgSUMZO4IXfPtWNnCQ4wQ6YJhikiDudVSGzCzezLYB
qlLwK8ZcmDF78fM42KqvDUzokMV/i3Ttm1R1jwIDAQABAoIBAQC0f0hSZ8HVgKqc
FECRCoB6KWd4/P3CyZ/9MUzNTnGiFSFcj1zbngXo+yty7vKJMaPcexmPNhzPNL2J
Ws+ZDHY7xFaVzk1tmYYuOu3/UnwPRAlpjtIO0vT/gjiNJwwzOisVnAn26b9DDDU6
X7UZQIKu5IefvSVOa9U0OYIlXAmGTZS0WdwYvvRX3H1+loWI81a18zo/weJKmJLM
WN5o4jFp/MfhBuTyIuugw8JwjJY+xHdXlKFDCuRMnhcCj8I7RL9e+7fqumlkQ2zn
XN4BTXFW7wS0kPnktj6i3sT8M5qr5UXzhpb/akjxoS8EE212gwNdcFsWQLU4ElLY
buSi5BihAoGBANuFqKyoGSyX34rJ6kVquYVQU+ZQSwZZvDnkR0B/d/glPerEwmmu
YX282eAQDaCEqHFTbMupoVEbgqMUS0DE8DajoJja68MeEPYwmZT8dbcF1RgnLxll
r645Q0ElOM8aU6uPcY1/Q1ZRTT4/DfRytsWm2Fcn6cSRBdIUZ4lmYSERAoGBAOdK
kOGLuFi1tpDpJrvPHzxx4om/v/hz1BnTDtYf0TuHOleb9FE7mfZoGV6//eBP7ecT
SmYlwkOPM8Zg3/S5v9CgcTDbvyafZSy+cHWiHpJidbFz5GpqsurYQtlk3OOIfS4j
AnjE+VKpy2+chSy1Y4s97UdqaGsCnRUxB8f+MiyfAoGBAKOnf1pIb4wZJSQ455gc
unYyrnmzPltbpsGZ96yT2wJ58TEGwtE6mZ+9nMg375DNlS33PdKPgZ4P3lJpnLiK
mXKChgWun7j0vzxqevThSXjKtlStUaWjc1d1hTgZ4cI0JFBwPf149OBy+B0BsQue
QbgUbJB9Rv+uBiLZ5149nwehAoGAJAaZSohYznh/V1L5lYdNdjzG7G3RmQFxqBQX
24JZNMx7aeoAxCZkdN0CFqARCz9n7vYyQHbhK2TCy8OLHrNQDU7wUovn2jw5ph6D
pc76tBJiAqHqkijMdYf54aK0RTydQvJxEB6eNyH7bgcTN0OJncanjtgkK4bcsNBS
RfRyHEECgYEA1oMq5XqtEQYyCJfKp0RQXLDTDk1QpNKaK+nHqUsgdpJGpmLNkUrK
A7IjdSkHooqbUtY+ouuE/WrZUbxQ4MFcfRnTb68VD2X7Z6ED0dWIGFm+QfAAqJWB
XCVFl0EOa/4XpgbHUjdItsytgSjGWyLnhKlaxejKUvqWr6MTJBB9O/w=
-----END RSA PRIVATE KEY-----

EOF
pre-k get cacert --common-name=ca </etc/kubernetes/pki/ca.key >/etc/kubernetes/pki/ca.crt

cat >/etc/kubernetes/pki/front-proxy-ca.key <<EOF
-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEA0dvl5dWTQPvh798NmOk4ni7uTrs8aHX1dkO0/V+zR/0chN/g
KOvK1yEwea9gXQM0QrMhtNMKl+yOekTPbDIqG0mCE3thXash5m2o9qmYC9/Aqp+A
gZ6mDamjqpTRso4AvVrLGcPMV7a1eEXKQKffdj1ZtK+ktnONtDTrda+mTpSLkw6v
cV2zJe8+8ZyIwKWJ5g5cu7oxaj/v2EkC1jQQc9gzhIfEqOm+nTxnSsAUfaSbHVd2
8l12J8R+SAkJs7JLsKpmQUiGk9omvO2equ3ZlFSiuU4kbxQj/16c8zSLnKRkOMY2
3vSeOHoFGC5CXnjcbkj0EFtsA6nmQvKCkRAgbQIDAQABAoIBAERc0u5k/ZwssXQ3
gDFwv+3fsefZ3JrW2khVVA857qgjzveGCnqqHSCpgiYRuF10XsDfo7pJCWlmOR+h
bMR9LvRGpOX9ykD+L4Pl8yWvJ4WutQ+f9/fBm6xVt6go5Iq68Yi9m+3ft6BXN5Rf
f7xCM2xMHf3bPdflmqK6nn1u48kyyUmMy1ZeycVRghZ2Df2xpb45c4C7IPaYkcTu
sJPd7ciq7VIooHhHNl0oPadnxwPceeiLYXnZvKdJs52su4sWWs+yToQDglZQmpkc
SbJUIbU+IuqgUIu5DyiJ/zs17gCXcB5gre1p16uOz+Phy8k8o2SvqAr2aYYvqp9k
t6y9zg0CgYEA+F474azcJxTF5OUD6AMjxpH6KLOe3GO2JXQ1RF97wLIT9n263Qhi
vXdbA0IAv3ghYvt+Z2qvvRY1Q6E2cP3ZN+KxKeC3TiNUyyacPkTmgLWhYfVOqM6z
NplOvlJV/W3qnQiS3PKg1Yf9yLSOGaQejM94ES5tPnw3Gk6YYL+X9GMCgYEA2E68
Rg+wd4x/5joEYygQogGmLOH6q/ob5h7bw1NT7/zyUhgUkZFBH8TgXJ2pLekcja1Z
mBIpj9rEeJ4c5C9hQzeXMXxBL+FDSY5MERfoeztxJMTe3WPmV8Dek5vnL77g6Kht
PbgAHtE9TP0Xui5EPVqY81ncYIhPO6z9CBqGqO8CgYAzNAI2YVO1vuOZb9lhUJxk
iJ3BHF7I4smfaRi+Ms0piczxyTPn852fn+akgkvzLUn8xQpnOahnXBe5DJhTrRHG
IrcRgiFoO4J0q04UzFGVAVz2/AKubIan3+1K8WCG8c0neKgGYwLjYrjgUtDk2l/t
+auwJxkgg608gC2L7JPgLwKBgQCyS5QdG+mVYRY7qy7anLe5EirrbAm3oB1G/cCf
rBvQAWusB3VM17Iagal+Lea5lSCYF392PeJMVUMFOS8PV46QXU4e2BDTapaQFt7U
aCSVD1YfvLcHPcUsKCpO+X8CeA/jNpF1Ain3PPmOcEASkvhkqjzQug1Q9Ip64ghf
mZ3NuQKBgDEEfVCbaKLzElFagXeEXRUykj6BLSDZlV8j3Z+D2P1u98KSKQlRb6uC
wMavxusG1UzsvpaSg3jLWB/jV4oLwMmXZHdybG4ei3DNuylumrUodqshK8b9aBal
v8tBxaaXhVJL8x/tqUGY5QazXFaRn2Z2A9DFPWq2QNSd3nz72vb0
-----END RSA PRIVATE KEY-----

EOF
pre-k get cacert --common-name=front-proxy-ca </etc/kubernetes/pki/front-proxy-ca.key >/etc/kubernetes/pki/front-proxy-ca.crt

chmod 600 /etc/kubernetes/pki/ca.key /etc/kubernetes/pki/front-proxy-ca.key

mkdir -p /etc/kubernetes/kubeadm

cat >/etc/kubernetes/kubeadm/base.yaml <<EOF
api:
  advertiseAddress: ""
  bindPort: 6443
apiServerExtraArgs:
  kubelet-preferred-address-types: InternalIP,ExternalIP
apiVersion: kubeadm.k8s.io/v1alpha1
certificatesDir: ""
cloudProvider: ""
etcd:
  caFile: ""
  certFile: ""
  dataDir: ""
  endpoints: null
  image: ""
  keyFile: ""
imageRepository: ""
kind: MasterConfiguration
kubernetesVersion: 1.9.0
networking:
  dnsDomain: cluster.local
  podSubnet: 192.168.0.0/16
  serviceSubnet: 10.96.0.0/12
nodeName: ""
token: ""
tokenTTL: 0s
unifiedControlPlaneImage: ""

EOF

pre-k merge master-config \
  --config=/etc/kubernetes/kubeadm/base.yaml \
  --apiserver-advertise-address=$(pre-k get public-ips --all=false) \
  --apiserver-cert-extra-sans=$(pre-k get public-ips --routable) \
  --apiserver-cert-extra-sans=$(pre-k get private-ips) \
  --apiserver-cert-extra-sans= \
  >/etc/kubernetes/kubeadm/config.yaml
kubeadm init --config=/etc/kubernetes/kubeadm/config.yaml --skip-token-print

kubectl apply \
  -f https://docs.projectcalico.org/v2.6/getting-started/kubernetes/installation/hosted/kubeadm/1.6/calico.yaml \
  --kubeconfig /etc/kubernetes/admin.conf

kubectl apply \
  -f https://raw.githubusercontent.com/pharmer/addons/release-1.13.1/kubeadm-probe/ds.yaml \
  --kubeconfig /etc/kubernetes/admin.conf

mkdir -p ~/.kube
sudo cp -i /etc/kubernetes/admin.conf ~/.kube/config
sudo chown $(id -u):$(id -g) ~/.kube/config

# kubectl taint nodes ${NODE_NAME} node.cloudprovider.kubernetes.io/uninitialized=true:NoSchedule --kubeconfig /etc/kubernetes/admin.conf
kubectl apply -f "https://raw.githubusercontent.com/pharmer/pharmer/ccm-fix/cloud/providers/digitalocean/cloud-control-manager.yaml" --kubeconfig /etc/kubernetes/admin.conf

#until [ $(kubectl get pods -n kube-system -l k8s-app=kube-dns -o jsonpath='{.items[0].status.phase}' --kubeconfig /etc/kubernetes/admin.conf) == "Running" ]
#do
#   echo '.'
#   sleep 5
#done

#until [ $(kubectl get pods -n kube-system -l app=cloud-controller-manager -o jsonpath='{.items[0].status.phase}' --kubeconfig /etc/kubernetes/admin.conf) == "Running" ]
#do
#   echo '.'
#   sleep 5
#done

#cat > /etc/systemd/system/kubelet.service.d/20-pharmer.conf <<EOF
#[Service]
#Environment="KUBELET_EXTRA_ARGS=--node-labels=cluster.pharmer.io/pool=master --cloud-provider=external "
#EOF

#NODE_NAME=$(uname -n)

#systemctl daemon-reload
#systemctl restart kubelet

# sleep 10
# reboot
