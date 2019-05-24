{{ define "credential-importing" }}
To get access on [packet](https://app.packet.net), `pharmer` needs credentials of `Packet`. To get the api key go to the **API Keys** section
under **my profile** option. Here you see the `Add an API key`, create and copy that key.

![packet-api-key](/docs/images/packet/packet-api-key.png)

From command line, run the following command and paste the api key.
```console
$ pharmer create credential packet
```
![packet-credential](/docs/images/packet/packet-credential.png)

Here, `pack` is the credential name, which must be unique within your storage.

To view credential file you can run:

```yaml
$ pharmer get credential packet -o yaml
apiVersion: v1alpha1
kind: Credential
metadata:
  creationTimestamp: 2017-11-02T11:31:34Z
  name: packet
spec:
  data:
    apiKey: <api-key>
    projectID: <project-id>
  provider: packet
```
Here,
 - `spec.data.projectID` is the packet project id
 - `spec.data.apiKey` is the access token that you provided which can be edited by following command:
```console
$ phrmer edit credential pack
```


To see the all credentials you need to run following command.

```console
$ pharmer get credentials
NAME         Provider       Data
packet       packet         projectID=6df2d99d...., apiKey=*****
```
You can also see the stored credential from the following location:
```console
~/.pharmer/store.d/$USER/credentials/pack.json
```

You can find other credential operations [here](/docs/credential.md)
{{ end }}


{{ define "tree" }}
```console
~/.pharmer/store.d/$USER/clusters/
/home/<user>/.pharmer/store.d/<user>/clusters/
├── {{ .Provider.ClusterName }}
│   ├── machine
│   │   ├── {{ .Provider.ClusterName }}-master-0.json
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

6 directories, 13 files
```
{{ end }}

{{ define "pending-cluster" }}
```yaml
$ pharmer get cluster {{ .Provider.ClusterName }} -o yaml
kind: Cluster
apiVersion: cluster.pharmer.io/v1beta1
metadata:
  name: {{ .Provider.ClusterName }}
  uid: a057bb8d-785a-11e9-901f-e0d55ee85d14
  generation: 1558066624400477400
  creationTimestamp: '2019-05-17T04:17:04Z'
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
          kind: PacketClusterProviderConfig
          apiVersion: Packetproviderconfig/v1alpha1
          metadata:
            creationTimestamp: 
    status: {}
  config:
    masterCount: 1
    cloud:
      cloudProvider: packet
      region: {{ .Provider.Location }}
      zone: {{ .Provider.Location }}
      instanceImage: ubuntu_16_04
      networkProvider: calico
      ccmCredentialName: pack
      sshKeyName: {{ .Provider.ClusterName }}-sshkey
    kubernetesVersion: {{ .KubernetesVersion }}
    caCertName: ca
    frontProxyCACertName: front-proxy-ca
    credentialName: pack
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
$ pharmer get cluster {{ .Provider.ClusterName }} -o yaml
kind: Cluster
apiVersion: cluster.pharmer.io/v1beta1
metadata:
  name: {{ .Provider.ClusterName }}
  uid: 157599fa-7861-11e9-9009-e0d55ee85d14
  generation: 1558069397870031400
  creationTimestamp: '2019-05-17T05:03:17Z'
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
          kind: PacketClusterProviderConfig
          apiVersion: Packetproviderconfig/v1alpha1
          metadata:
            creationTimestamp: 
    status:
      apiEndpoints:
      - host: 147.75.192.173
        port: 6443
      providerStatus:
        apiVersion: Packetproviderconfig/v1alpha1
        kind: PacketClusterProviderConfig
        metadata:
          creationTimestamp: 
  config:
    masterCount: 1
    cloud:
      cloudProvider: packet
      region: {{ .Provider.Location }}
      zone: {{ .Provider.Location }}
      instanceImage: ubuntu_16_04
      networkProvider: calico
      ccmCredentialName: pack
      sshKeyName: {{ .Provider.ClusterName }}-sshkey
    kubernetesVersion: {{ .KubernetesVersion }}
    caCertName: ca
    frontProxyCACertName: front-proxy-ca
    credentialName: pack
    apiServerExtraArgs:
      kubelet-preferred-address-types: InternalIP,ExternalIP
status:
  phase: Pending
  cloud:
    sshKeyExternalID: 35f3eff3-2148-4384-8c71-8ab63e4c86b6
    loadBalancer:
      dns: ''
      ip: ''
      port: 0
```
{{ end }}

{{ define "get-nodes" }}
```console
$ kubectl get nodes
{{ .Provider.ClusterName }}-master-0        Ready    master   29m   {{ .KubernetesVersion }}
{{ .MachinesetName }}   Ready    node     13m   {{ .KubernetesVersion }}
```
{{ end }}

{{ define "get-machines" }}
```console
$ kubectl get machines
NAME               AGE
{{ .MachinesetName }}   1m
{{ .Provider.ClusterName }}-master-0        2m

$ kubectl get machinesets
NAME               AGE
{{ .MachinesetName }}   2m
```
{{ end }}

{{ define "worker-machine" }}
```yaml
kind: Machine
apiVersion: cluster.k8s.io/v1alpha1
metadata:
  name: worker-1
  labels:
    cluster.k8s.io/cluster-name: {{ .Provider.ClusterName }}
    node-role.kubernetes.io/master: ''
    set: node
spec:
  providerSpec:
    value:
      kind: PacketClusterProviderConfig
      apiVersion: Packetproviderconfig/v1alpha1
      plan: {{ .Provider.NodeSpec.SKU }}
      type: Regular
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
          kind: PacketClusterProviderConfig
          apiVersion: Packetproviderconfig/v1alpha1
          plan: {{ .Provider.NodeSpec.SKU }}
          type: Regular
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
          kind: PacketClusterProviderConfig
          apiVersion: Packetproviderconfig/v1alpha1
          plan: {{ .Provider.NodeSpec.SKU }}
          type: Regular
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
  uid: 379d4d7f-77c1-11e9-b997-e0d55ee85d14
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

{{ define "master-machine" }}
{{ end }}
