package digitalocean

import (
  "testing"
  kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1alpha3"
  kubeadmscheme "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/scheme"
  "fmt"
)
func TestStartup(t *testing.T) {
  data :=`
apiEndpoint:
  advertiseAddress: ""
  bindPort: 6443
apiVersion: kubeadm.k8s.io/v1alpha3
kind: InitConfiguration
nodeRegistration:
  kubeletExtraArgs:
    cloud-provider: external
    node-labels: cloud.appscode.com/pool=master

---
apiServerExtraArgs:
  kubelet-preferred-address-types: InternalIP,ExternalIP
apiVersion: kubeadm.k8s.io/v1alpha3
auditPolicy:
  logDir: ""
  path: ""
certificatesDir: ""
clusterName: d1120a
controlPlaneEndpoint: ""
etcd: {}
imageRepository: ""
kind: ClusterConfiguration
kubernetesVersion: v1.12.0
networking:
  dnsDomain: cluster.local
  podSubnet: 192.168.0.0/16
  serviceSubnet: 10.96.0.0/12
unifiedControlPlaneImage: ""
`
fmt.Println(data)
cfg := &kubeadmapi.InitConfiguration{}
kubeadmscheme.Scheme.Default(cfg)
//kubeadmapi.SetDefaults_InitConfiguration(cfg)
fmt.Println(cfg)

}
