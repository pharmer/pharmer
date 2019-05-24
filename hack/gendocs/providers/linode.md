{{ define "credential-importing" }}

To get access on [Linode](https://www.linode.com/), `pharmer` needs credentials of `Linode`. To get the api key go to the **API Keys** section
under **my profile** option. Here you see the `Add an API key`, create and copy that key.

![linode-api-key](/docs/images/linode/linode-api-key.jpg)

From command line, run the following command and paste the api key.
```console
$ pharmer create credential linode
```
![linode-credential](/docs/images/linode/linode-credential.png)

Here, `linode` is the credential name, which must be unique within your storage.

To view credential file you can run:
```yaml
~ $ pharmer get credentials linode -o yaml
apiVersion: v1alpha1
kind: Credential
metadata:
  creationTimestamp: 2017-11-01T04:47:56Z
  name: linode
spec:
  data:
    token: <your token>
  provider: linode

```
Here, `spec.data.token` is the access token that you provided which can be edited by following command:
```console
$ phrmer edit credential linode
```

To see the all credentials you need to run following command:

```console
$ pharmer get credentials
NAME         Provider       Data
linode       linode         token=*****
```

You can also see the stored credential from the following location:
```console
~/.pharmer/store.d/$USER/credentials/linode.json
```

You can find other credential operations [here](/docs/credential.md)
{{ end }}


{{ define "tree" }}
```console
~/.pharmer/store.d/$USER/clusters/
├── {{ .Provider.ClusterName }}
│   ├── machine
│   │   ├── {{ .Provider.ClusterName }}-master-0.json
│   │   ├── {{ .Provider.ClusterName }}-master-1.json
│   │   └── {{ .Provider.ClusterName }}-master-2.json
│   ├── machineset
│   │   └── {{ .MachinesetName }}.json
│   ├── pki
│   │   ├── ca.crt
│   │   ├── ca.key
│   │   ├── etcd
│   │   │   ├── ca.crt
│   │   │   └── ca.key
│   │   ├── front-proxy-ca.crt
│   │   ├── front-proxy-ca.key
│   │   ├── sa.crt
│   │   └── sa.key
│   └── ssh
│       ├── id_{{ .Provider.ClusterName }}-sshkey
│       └── id_{{ .Provider.ClusterName }}-sshkey.pub
└── {{ .Provider.ClusterName }}.json

6 directories, 15 files
```
{{ end }}

{{ define "pending-cluster" }}
```yaml
$ pharmer get cluster {{ .Provider.ClusterName }} -o yaml
kind: Cluster
apiVersion: cluster.pharmer.io/v1beta1
metadata:
  name: {{ .Provider.ClusterName }}
  uid: 47ed5e2a-7856-11e9-8051-e0d55ee85d14
  generation: 1558064758076986000
  creationTimestamp: '2019-05-17T03:45:58Z'
spec:
  clusterApi:
    kind: Cluster
    apiVersion: cluster.k8s.io/v1alpha1
    metadata:
      name: {{ .Provider.ClusterName }}
      namespace: default
      creationTimestamp: 
    spec:
      clusterNetwork:
        services:
          cidrBlocks:
          - 10.96.0.0/12
        pods:
          cidrBlocks:
          - 192.168.0.0/16
        serviceDomain: cluster.local
      providerSpec:
        value:
          kind: LinodeClusterProviderConfig
          apiVersion: linodeproviderconfig/v1alpha1
          metadata:
            creationTimestamp: 
    status: {}
  config:
    masterCount: 1
    cloud:
      cloudProvider: linode
      region: {{ .Provider.Location }}
      zone: {{ .Provider.Location }}
      instanceImage: linode/ubuntu16.04lts
      networkProvider: calico
      ccmCredentialName: linode
      sshKeyName: {{ .Provider.ClusterName }}-sshkey
      linode:
        rootPassword: 9GPOgQZbSZ4gwxT0
    kubernetesVersion: {{ .KubernetesVersion }}
    caCertName: ca
    frontProxyCACertName: front-proxy-ca
    credentialName: linode
    apiServerExtraArgs:
      kubelet-preferred-address-types: InternalIP,ExternalIP
status:
  phase: Pending
  cloud:
    loadBalancer:
      dns: ''
      ip: ''
      port: 0
```
{{ end }}


{{ define "ready-cluster" }}
```yaml
pharmer get cluster {{ .Provider.ClusterName }} -o yaml
---
kind: Cluster
apiVersion: cluster.pharmer.io/v1beta1
metadata:
  name: {{ .Provider.ClusterName }}
  uid: 47ed5e2a-7856-11e9-8051-e0d55ee85d14
  generation: 1558065344523630600
  creationTimestamp: '2019-05-17T03:45:58Z'
spec:
  clusterApi:
    kind: Cluster
    apiVersion: cluster.k8s.io/v1alpha1
    metadata:
      name: {{ .Provider.ClusterName }}
      namespace: default
      creationTimestamp: 
    spec:
      clusterNetwork:
        services:
          cidrBlocks:
          - 10.96.0.0/12
        pods:
          cidrBlocks:
          - 192.168.0.0/16
        serviceDomain: cluster.local
      providerSpec:
        value:
          apiVersion: linodeproviderconfig/v1alpha1
          kind: LinodeClusterProviderConfig
          metadata:
            creationTimestamp: 
    status:
      apiEndpoints:
      - host: 96.126.119.162
        port: 6443
      providerStatus:
        metadata:
          creationTimestamp: 
        network:
          apiServerLb:
            client_conn_throttle: 20
            hostname: nb-96-126-119-162.dallas.nodebalancer.linode.com
            id: 47809
            ipv4: 96.126.119.162
            ipv6: 2600:3c00:1::607e:77a2
            label: {{ .Provider.ClusterName }}-lb
            region: {{ .Provider.Location }}
            tags: []
  config:
    masterCount: 3
    cloud:
      cloudProvider: linode
      region: {{ .Provider.Location }}
      zone: {{ .Provider.Location }}
      instanceImage: linode/ubuntu16.04lts
      networkProvider: calico
      ccmCredentialName: linode
      sshKeyName: {{ .Provider.ClusterName }}-sshkey
      linode:
        rootPassword: 9GPOgQZbSZ4gwxT0
        kernelId: linode/latest-64bit
    kubernetesVersion: {{ .KubernetesVersion }}
    caCertName: ca
    frontProxyCACertName: front-proxy-ca
    credentialName: linode
    apiServerExtraArgs:
      kubelet-preferred-address-types: InternalIP,ExternalIP
status:
  phase: Ready
  cloud:
    loadBalancer:
      dns: ''
      ip: 96.126.119.162
      port: 6443
```
{{ end }}


{{ define "get-nodes" }}
Now you can run `kubectl get nodes` and verify that your kubernetes 1.13.5 is running.
```console
$ kubectl get nodes
NAME                       STATUS   ROLES    AGE     VERSION
{{ .Provider.ClusterName }}-master-0                Ready    master   6m21s   {{ .KubernetesVersion }}
{{ .Provider.ClusterName }}-master-1                Ready    master   3m10s   {{ .KubernetesVersion }}
{{ .Provider.ClusterName }}-master-2                Ready    master   2m7s    {{ .KubernetesVersion }}
{{ .MachinesetName }}-5pft6   Ready    node     56s     {{ .KubernetesVersion }}
```
{{ end }}


{{ define "get-machines" }}
```console
$ kubectl get machines
NAME                       AGE
{{ .MachinesetName }}-pft6v   4m
{{ .Provider.ClusterName }}-master-0                4m
{{ .Provider.ClusterName }}-master-1                4m
{{ .Provider.ClusterName }}-master-2                4m

$ kubectl get machinesets
NAME                 AGE
{{ .MachinesetName }}   5m
```
{{ end }}

{{ define "master-machine" }}
```yaml
kind: Machine
apiVersion: cluster.k8s.io/v1alpha1
metadata:
  name: l1-master-3
  labels:
    cluster.k8s.io/cluster-name: l1
    node-role.kubernetes.io/master: ''
    set: controlplane
spec:
  providerSpec:
    value:
      kind: LinodeClusterProviderConfig
      apiVersion: linodeproviderconfig/v1alpha1
      roles:
      - Master
      region: {{ .Provider.Location }}
      type: {{ .Provider.NodeSpec.SKU }}
      image: linode/ubuntu16.04lts
      pubkey: ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC+e0+nNjre2JoHDySl1HiWsWmwLyIs1+KbQIM/15JKgQLA4PpkrVwOQkxjZOA+ozBoBgilAFLbca5ERUXeMz8nMoF1+JxwUoqSAeeA7PB1ZyU78r7c0nb1OMcMxituHZnpwtXxbsAAWMXBiEQx8IrlttJrlaD/7IP+jlSzwCyLOAVU/86euC+2bVi6SxnbVNy+POmrwncAx1VXHLP2o1zM9L+ENhYwR1YNfBeQse1zqSafgxEFU9SCrJGbnq6mJNS3U/dVg96Aj4QCYuC8wB7Nmca7U7/3EjTa+rXHzc5g1lqcHI2s26niK34kPLiu8vxp9k2Gkw2Me88Z4J60dScD
  versions:
    kubelet: {{ .KubernetesVersion }}
    controlPlane: {{ .KubernetesVersion }}
```
{{ end }}

{{ define "worker-machine" }}
```yaml
kind: Machine
apiVersion: cluster.k8s.io/v1alpha1
metadata:
  name: l1-master-0
  creationTimestamp: '2019-05-17T03:45:59Z'
  labels:
    cluster.k8s.io/cluster-name: l1
    node-role.kubernetes.io/node: ''
    set: node
spec:
  providerSpec:
    value:
      kind: LinodeClusterProviderConfig
      apiVersion: linodeproviderconfig/v1alpha1
      roles:
      - Node
      region: {{ .Provider.Location }}
      type: {{ .Provider.NodeSpec.SKU }}
      image: linode/ubuntu16.04lts
      pubkey: ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC+e0+nNjre2JoHDySl1HiWsWmwLyIs1+KbQIM/15JKgQLA4PpkrVwOQkxjZOA+ozBoBgilAFLbca5ERUXeMz8nMoF1+JxwUoqSAeeA7PB1ZyU78r7c0nb1OMcMxituHZnpwtXxbsAAWMXBiEQx8IrlttJrlaD/7IP+jlSzwCyLOAVU/86euC+2bVi6SxnbVNy+POmrwncAx1VXHLP2o1zM9L+ENhYwR1YNfBeQse1zqSafgxEFU9SCrJGbnq6mJNS3U/dVg96Aj4QCYuC8wB7Nmca7U7/3EjTa+rXHzc5g1lqcHI2s26niK34kPLiu8vxp9k2Gkw2Me88Z4J60dScD
  versions:
    kubelet: {{ .KubernetesVersion }}
```
{{ end }}

{{ define "machineset" }}
```yaml
kind: MachineSet
apiVersion: cluster.k8s.io/v1alpha1
metadata:
  name: {{ .MachinesetName }}
spec:
  replicas: 1
  selector:
    matchLabels:
      cluster.k8s.io/cluster-name: {{ .Provider.ClusterName }}
      cluster.pharmer.io/mg: {{ .Provider.NodeSpec.SKU }}
  template:
    metadata:
      labels:
        cluster.k8s.io/cluster-name: {{ .Provider.ClusterName }}
        cluster.pharmer.io/cluster: {{ .Provider.ClusterName }}
        cluster.pharmer.io/mg: {{ .Provider.NodeSpec.SKU }}
        node-role.kubernetes.io/node: ''
        set: node
    spec:
      providerSpec:
        value:
          kind: LinodeClusterProviderConfig
          apiVersion: linodeproviderconfig/v1alpha1
          roles:
          - Node
          region: {{ .Provider.Location }}
          type: {{ .Provider.NodeSpec.SKU }}
          image: linode/ubuntu16.04lts
          pubkey: ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC+e0+nNjre2JoHDySl1HiWsWmwLyIs1+KbQIM/15JKgQLA4PpkrVwOQkxjZOA+ozBoBgilAFLbca5ERUXeMz8nMoF1+JxwUoqSAeeA7PB1ZyU78r7c0nb1OMcMxituHZnpwtXxbsAAWMXBiEQx8IrlttJrlaD/7IP+jlSzwCyLOAVU/86euC+2bVi6SxnbVNy+POmrwncAx1VXHLP2o1zM9L+ENhYwR1YNfBeQse1zqSafgxEFU9SCrJGbnq6mJNS3U/dVg96Aj4QCYuC8wB7Nmca7U7/3EjTa+rXHzc5g1lqcHI2s26niK34kPLiu8vxp9k2Gkw2Me88Z4J60dScD
      versions:
        kubelet: {{ .KubernetesVersion }}
```
{{ end }}



{{ define "machinedeployment" }}
```yaml
kind: MachineDeployment
apiVersion: cluster.k8s.io/v1alpha1
metadata:
  name: {{ .MachinesetName }}
spec:
  replicas: 1
  selector:
    matchLabels:
      cluster.k8s.io/cluster-name: {{ .Provider.ClusterName }}
      cluster.pharmer.io/mg: {{ .Provider.NodeSpec.SKU }}
  template:
    metadata:
      labels:
        cluster.k8s.io/cluster-name: {{ .Provider.ClusterName }}
        cluster.pharmer.io/cluster: {{ .Provider.ClusterName }}
        cluster.pharmer.io/mg: {{ .Provider.NodeSpec.SKU }}
        node-role.kubernetes.io/node: ''
        set: node
    spec:
      providerSpec:
        value:
          kind: LinodeClusterProviderConfig
          apiVersion: linodeproviderconfig/v1alpha1 
          roles:
          - Node
          region: {{ .Provider.Location }}
          type: {{ .Provider.NodeSpec.SKU }}
          image: linode/ubuntu16.04lts
          pubkey: ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC+e0+nNjre2JoHDySl1HiWsWmwLyIs1+KbQIM/15JKgQLA4PpkrVwOQkxjZOA+ozBoBgilAFLbca5ERUXeMz8nMoF1+JxwUoqSAeeA7PB1ZyU78r7c0nb1OMcMxituHZnpwtXxbsAAWMXBiEQx8IrlttJrlaD/7IP+jlSzwCyLOAVU/86euC+2bVi6SxnbVNy+POmrwncAx1VXHLP2o1zM9L+ENhYwR1YNfBeQse1zqSafgxEFU9SCrJGbnq6mJNS3U/dVg96Aj4QCYuC8wB7Nmca7U7/3EjTa+rXHzc5g1lqcHI2s26niK34kPLiu8vxp9k2Gkw2Me88Z4J60dScD
      versions:
        kubelet: {{ .KubernetesVersion }}
```
{{ end }}

{{ define "deleted-cluster" }}
```yaml
$ pharmer get cluster {{ .Provider.ClusterName }} -o yaml
kind: Cluster
apiVersion: cluster.pharmer.io/v1beta1
metadata:
  name: {{ .Provider.ClusterName }}
  uid: 379d4d7f-77c1-11e9-b997-e0d55ee85li4
  generation: 1558000735696016100
  creationTimestamp: '2019-05-16T09:58:55Z'
  deletionTimestamp: '2019-05-16T10:38:54Z'
...
...
status:
  phase: Deleting
...
...
```
{{ end }}

{{ define "ssh" }}
{{ end }}

