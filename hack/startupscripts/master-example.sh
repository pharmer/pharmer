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
MIIEpAIBAAKCAQEAv577t9QJEdHA1ywCGf1/PPl4aVfSRpw5CaOAYZcqAMMXaqc8
HAG/JaP3/N0O+UOX3BsJAjfyLLEUxDDNgVmn1PcSOFye+BpmfdbJgBxwqvQcsZxS
MyJrs8hYVMAZCI9Hqgc9PtNZht6ZnSXI+gmzwe13FqvoldzTwYaV8mmacANqGaqy
sxMclzZ/1P0D2QUE3TeJGe17TjRpljYr1Ye07b1JQWHXFYtwQ+x69YMnuHuNEVXb
bePZBiIDd3oAjNlDGWNKcNx05OCbJMJGwUHuThQZ0djaEktYYcgM9PUSMsdkTqgd
DsyROrVoPVkv97+QTEcax7854a6OMKggbpOl+wIDAQABAoIBAHElxWj00S4udvoX
SF5kaz9r54f8jXOYR63DV8lIY0rz593YRgwbk+z79zDerzrMiN01MYqX0m5FWgfG
8XIRqKJ+CcoyFsRVgnOH4o+9O2IWpuLRdVAwYPxVrqLMk3uUbzKc97MjRVf4FSjP
6O6L0BMQzyMZ+0qSjI9Xky8C4GiG9VwIpmN3zXKna4yTZZuVKA6llVGJmEOLCdrG
gDiYtfo/qF6q2DO+l6xe0dPdTOH7tOJkDOzc0+1LeJ9LumMUpV1iKFMrAXH9Jx9R
WFEuldliOkAJflyc3NGwkSdhMiqmn5UpaBQDSRa5XhiUKz2Qrk6aFYZILRiMAJpd
dr+GYBECgYEAzfYiGoYZGGnZUy85MHu5tRTQGfyFkYhDaLzPvQoH3C0YhqXedfFz
vWbS21ViiyXC2Vx0RaHDM9g0WBBsIzYQERg1PyjkLWv1vpMs+ci7K280cDC9/Ron
sBJgilePiJOFEA4yaWjXOin9aFPA0Wg+X2AVw3dQWYn5czxkiZ47IncCgYEA7izw
xRRrVLIq5YT/5EDlaViDxjs4xg5e405vqURlHfFIKv7GWOhZ1FFQn0AhqkWHqdX6
Zavi041TTtlZ+5DSbWj6P3sEpVL0aCLCl3S3dNmAZB0atwlKlfgFGFHwm6gQw/kP
g1+aGB0gRz1DnW81if8EcthC0oZnhr3s3jaSVZ0CgYBeEDpVGMdfUgBSMHATB36y
B/Ze1+h8pdn7fLf3oxwh32qjHB/0h7iPWOWWDHH8ENvjf4kOwCkxhV2qlp18m1VN
KVwwI1HKuNK8HeVdkuKAnMI7NteeP5K+pDX5GLJR8uXDhmhZoesiAklcm1ulh9Fs
p6po4hYNPqlxirRz8ZMaUwKBgQCNXCz2u5zMYwakvOBnp8DBWqCzWdjjbAoTXS1W
yFo/gEI2CorMn/MY2b5BFn4koinXukFocLEqoFmOleAbOCQ8fa7xWGE0gly/Jcpa
vBJajvDt+nwtoJ0dD1xux8tHh2OT/NGhRm+d2kObJJhp62RaZ/pK82INs2nkhfr9
FGSb3QKBgQC0tPSSXOjleXR+Al2q3yfzcCzqID4ug5rpPtv6mH1ZzA0OJ5G5EjOz
eIwd43aezReRMWBVOutMV24L7TV/jBWdF5PpRz03EnDDKfatitIVXyTLX7LdSdlo
JQljWIvAW82/Ut06BYKjl2YClNkXTFnp/tSynyUvjs6ZunDJEE3LLQ==
-----END RSA PRIVATE KEY-----

EOF
pre-k get cacert --common-name=ca < /etc/kubernetes/pki/ca.key > /etc/kubernetes/pki/ca.crt

cat > /etc/kubernetes/pki/front-proxy-ca.key <<EOF
-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEA2KngAzHeVFeLld/ACMOdpddlSKW0s47OapYqUEUBp4kZvRDg
+oYbsMDLM2xfPqzEYA/qYFNEzZTMbSS9LerJXkCuJpsLrBRw+2s29VExge5zAq7n
IPS7OIkk1jJxUxj6n5EG72Vz03nHXsX+aiqbHTmGiliXusa8t1F66yMD4CdHZGSD
w30V8TVWG7m8vUxLQykFd0FjBwuRVibB4Xe8gMBNLWEMeLgwEIfhiaN9Gdt8J5k+
I+yzhWsbIxjvHNSJvfJSpnlrsjAcbe2CWf7OLkI0vmjcv6XIKC4PblH+HCiAKUSe
k5HWFbmqEjaQGqUFq3Hc5TkPw8Ab7qkn1QLXpQIDAQABAoIBACVmhZ3nkNp8VkMj
4bFhmygSm5BD0oKgZy9tBpbeop96AjBp5nw4hDUpwqn4ziQyjJ+Mf8fq99iJFBTP
H7z6Z5JWlfliEEy/JpWv90e3oVUthTr0WD+PH3Zt7ibvhDJ1AGZlEY5ns+vQfML2
aKmI+0e7M2dUTbTXM6JtdUt7uuKJc+5HZLl/18Q/MFpFksYbSE1W7njqcMBhqG+5
AcBktFo2UUuRuzxYdF4/6ammwnNUQABcKRh22THYAD3b7rUAPrOP/BMBXvAzfq7w
2u51/fjQgKn61lLTE9674dwALTe6fqMYH1NVY90j74lUJtBKDwpF/nrPOmgZB55v
lflW7gECgYEA74f6f8JD4Z1rrzGrvIuiXjVP5r9Hke/KdjliDH8gcErg4CbAcGQM
vRq3fxr74XfHpiPIHeFkypy0oSmNf1ZWHThYoEMSPYsTBBOw/XtK+oz8hGMbjKGG
gSimJsKBNimP52RgYyc+DzY8YQbdIqPU6GC4phanD3gbbLXIfQ25siUCgYEA549m
nwmPhwLSn0FbwJ09byIYfCgUwKqU1jLvQzw5RmWPJmGYPc6pwiUrd7DKC/XskUb/
jNGjDKtoJ/iXH3eWk1t4+9d6mNuAPBLUitwC55fNeURj6Yx/HgtTfLBiwTTWUeen
pS4U7p6vWP3eniD2SCS0zpsHSBEXtpaxLrqX14ECgYBn7kROsufpOPjEZ3TwtlWZ
MuCcalqPg9ABRBy6914H2zV/jNDq8z5jvvfnernXtrU17UijTm0BTMVDOFhg7AEp
ZI1v3CpJ8dIqbrDZC2oZFJtsheQqPNuzpwOQxcxyx83cxLzdyOUwjIPeRbAlm5iB
y5XS2vlZyO99V9LKD+6G0QKBgG+iP5wOdOZm4vmp/bznUVXBf0Jj5FVaJ3T4i0bD
wu0ASzn6tPWfK1IQr1r9nPqVLd5/9vWBn3SdMhKLEvg1pB8Ya0QmhtEoiTLs9RCY
rsHN+l4rLWvLswDd/vrO4l04xtYnsze79pVvZSOnCGr9gx4WrCAqtyD0NJVvIq5f
gJKBAoGBAKIIVKpYUJVgldfZyJxXUMQeQNw3LpczDUqQVZHnJDgJM1bftKM1A+BQ
mepILgNWkUlAyc4Bn79UW2/sPePKs6BaCEJXEI6MEkYZtgXhgsvp5pM0hej/QHC+
aoTtTdgAq41V91wbu7QryBWDct2lJIAMgQJaYyoSZNtuDU0Ryb0N
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
