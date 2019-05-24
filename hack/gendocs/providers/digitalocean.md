{{ define "credential-importing" }}
### Credential importing

Get an access token by following the [guide](https://www.digitalocean.com/community/tutorials/how-to-use-the-digitalocean-api-v2#how-to-generate-a-personal-access-token) and pass to it pharmer.

```console
$ pharmer create credential do
Choose a Cloud provider: DigitalOcean
Personal Access Token
****************************
```

To view credential file you can run:
```yaml
apiVersion: v1alpha1
kind: Credential
metadata:
  creationTimestamp: 2017-10-03T05:13:07Z
  name: do
spec:
  data:
    token: <token>
  provider: digitalOcean
```
Here,
- `spec.data.token` is the access token that you provided which can be edited by following command:
```console
$ pharmer edit credential do
```

To see the all credentials you need to run following command.

```console
$ pharmer get credentials
NAME         Provider       Data
do           digitalocean   token=*****
```
You can also see the stored credential from the following location:
```console
~/.pharmer/store.d/$USER/credentials/do.json
```

You can find other credential operations [here](/docs/credential.md)

{{ end }}

{{ define "tree" }}

```console
$ tree ~/.pharmer/store.d/$USER/clusters/
/home/<user>/.pharmer/store.d/<user>/clusters/
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
  uid: 379d4d7f-77c1-11e9-b997-e0d55ee85d14
  generation: 1558000735696016100
  creationTimestamp: '2019-05-16T09:58:55Z'
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
          kind: DigitalOceanProviderConfig
          apiVersion: digitaloceanproviderconfig/v1alpha1
          metadata:
            creationTimestamp:
    status: {}
  config:
    {{ .Provider.MasterNodeCount }}
    cloud:
      cloudProvider: digitalocean
      region: {{ .Provider.Location }}
      zone: {{ .Provider.Location }}
      instanceImage: ubuntu-18-04-x64
      networkProvider: calico
      sshKeyName: {{ .Provider.ClusterName }}-sshkey
    kubernetesVersion: {{ .KubernetesVersion }}
    caCertName: ca
    frontProxyCACertName: front-proxy-ca
    credentialName: do
    apiServerExtraArgs:
      kubelet-preferred-address-types: ExternalDNS,ExternalIP,InternalIP
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
  uid: 379d4d7f-77c1-11e9-b997-e0d55ee85d14
  generation: 1558000735696016100
  creationTimestamp: '2019-05-16T09:58:55Z'
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
          kind: DigitalOceanProviderConfig
          apiVersion: digitaloceanproviderconfig/v1alpha1
          metadata:
            creationTimestamp:
    status:
      apiEndpoints:
      - host: 138.197.226.237
        port: 6443
      providerStatus:
        apiServerLb:
          algorithm: least_connections
          created_at: '2019-05-16T10:06:25Z'
          forwarding_rules:
          - entry_port: 6443
            entry_protocol: tcp
            target_port: 6443
            target_protocol: tcp
          health_check:
            check_interval_seconds: 3
            healthy_threshold: 5
            port: 6443
            protocol: tcp
            response_timeout_seconds: 5
            unhealthy_threshold: 3
          id: d478fb9f-2bf2-4884-b9df-37c1e0e1f877
          ip: 138.197.226.237
          name: {{ .Provider.ClusterName }}-lb
          region: {{ .Provider.Location }}
          status: active
          sticky_sessions:
            type: none
        metadata:
          creationTimestamp:
  config:
    {{ .Provider.MasterNodeCount }}
    cloud:
      cloudProvider: digitalocean
      region: {{ .Provider.Location }}
      zone: {{ .Provider.Location }}
      instanceImage: ubuntu-18-04-x64
      networkProvider: calico
      sshKeyName: {{ .Provider.ClusterName }}-sshkey
    kubernetesVersion: {{ .KubernetesVersion }}
    caCertName: ca
    frontProxyCACertName: front-proxy-ca
    credentialName: do
    apiServerExtraArgs:
      kubelet-preferred-address-types: ExternalDNS,ExternalIP,InternalIP
status:
  phase: Ready
  cloud:
    sshKeyExternalID: '24595729'
    loadBalancer:
      dns: ''
      ip: 138.197.226.237
      port: 6443
```
{{ end }}

{{ define "get-nodes" }}
```console
$ kubectl get nodes

NAME             STATUS   ROLES    AGE   VERSION
{{ .MachinesetName }}-p2c7m   Ready    node     13m   {{ .KubernetesVersion }}
{{ .Provider.ClusterName }}-master-0      Ready    master   29m   {{ .KubernetesVersion }}
{{ .Provider.ClusterName }}-master-1      Ready    master   14m   {{ .KubernetesVersion }}
{{ .Provider.ClusterName }}-master-2      Ready    master   13m   {{ .KubernetesVersion }}
```
{{ end }}


{{ define "get-machines" }}
```console
$ kubectl get machines
NAME             AGE
{{ .MachinesetName }}-p2c7m   1m
{{ .Provider.ClusterName }}-master-0      2m
{{ .Provider.ClusterName }}-master-1      2m
{{ .Provider.ClusterName }}-master-2      2m

$ kubectl get machinesets
NAME       AGE
{{ .MachinesetName }}   2m
```
{{ end }}


{{ define "master-machine" }}
```yaml
kind: Machine
apiVersion: cluster.k8s.io/v1alpha1
metadata:
  name: {{ .Provider.ClusterName }}-master-3
  creationTimestamp: '2019-05-16T09:58:56Z'
  labels:
    cluster.k8s.io/cluster-name: {{ .Provider.ClusterName }}
    node-role.kubernetes.io/master: ''
    set: controlplane
spec:
  metadata:
    creationTimestamp:
  providerSpec:
    value:
      kind: DigitalOceanProviderConfig
      apiVersion: digitaloceanproviderconfig/v1alpha1
      creationTimestamp:
      region: {{ .Provider.Location }}
      size: {{ .Provider.NodeSpec.SKU }}
      image: ubuntu-18-04-x64
      tags:
      - KubernetesCluster:{{ .Provider.ClusterName }}
      sshPublicKeys:
      - ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDa7/godjDMz8zAY0dMjujPQGoN/dUgH6b4E9WOUIEXQH9lbZx7yXRJ/COPHvXVUqVjNO8BkNHWGrDDr7Ozq8yfKz4xTFMRM9IsXFwopsC5ijd7XiHnwjJsDWAMbLje1AOL4MDvmSuVJUmr6ZuaAZUTg3zRredBxdiw0nj1pQEuHZ29DVmmoedM2CDxGMwR+sFOvgvkW4pJLUbq3uXxFN1z2t/djyO+YENHe3BRJ2jA9SMi+7KrN3Z3N09r6CtdeSRm/m3GDsreyWDJRsUJ9w1XGQc2qYcpDBycUPjBfD2nLeTxlZp5JAu74P+QTbmghoT9MudOqZE+XLkLE9saxozn
      private_networking: true
      monitoring: true
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
  name: worker-1
  labels:
    cluster.k8s.io/cluster-name: {{ .Provider.ClusterName }}
    node-role.kubernetes.io/master: ''
    set: node
spec:
  metadata:
    creationTimestamp:
  providerSpec:
    value:
      kind: DigitalOceanProviderConfig
      apiVersion: digitaloceanproviderconfig/v1alpha1
      region: {{ .Provider.Location }}
      size: {{ .Provider.NodeSpec.SKU }}
      image: ubuntu-18-04-x64
      tags:
      - KubernetesCluster:{{ .Provider.ClusterName }}
      sshPublicKeys:
      - ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDa7/godjDMz8zAY0dMjujPQGoN/dUgH6b4E9WOUIEXQH9lbZx7yXRJ/COPHvXVUqVjNO8BkNHWGrDDr7Ozq8yfKz4xTFMRM9IsXFwopsC5ijd7XiHnwjJsDWAMbLje1AOL4MDvmSuVJUmr6ZuaAZUTg3zRredBxdiw0nj1pQEuHZ29DVmmoedM2CDxGMwR+sFOvgvkW4pJLUbq3uXxFN1z2t/djyO+YENHe3BRJ2jA9SMi+7KrN3Z3N09r6CtdeSRm/m3GDsreyWDJRsUJ9w1XGQc2qYcpDBycUPjBfD2nLeTxlZp5JAu74P+QTbmghoT9MudOqZE+XLkLE9saxozn
      private_networking: true
      monitoring: true
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
          kind: DigitalOceanProviderConfig
          apiVersion: digitaloceanproviderconfig/v1alpha1
          region: {{ .Provider.Location }}
          size: {{ .Provider.NodeSpec.SKU }}
          image: ubuntu-18-04-x64
          tags:
          - KubernetesCluster:{{ .Provider.ClusterName }}
          sshPublicKeys:
          - ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDa7/godjDMz8zAY0dMjujPQGoN/dUgH6b4E9WOUIEXQH9lbZx7yXRJ/COPHvXVUqVjNO8BkNHWGrDDr7Ozq8yfKz4xTFMRM9IsXFwopsC5ijd7XiHnwjJsDWAMbLje1AOL4MDvmSuVJUmr6ZuaAZUTg3zRredBxdiw0nj1pQEuHZ29DVmmoedM2CDxGMwR+sFOvgvkW4pJLUbq3uXxFN1z2t/djyO+YENHe3BRJ2jA9SMi+7KrN3Z3N09r6CtdeSRm/m3GDsreyWDJRsUJ9w1XGQc2qYcpDBycUPjBfD2nLeTxlZp5JAu74P+QTbmghoT9MudOqZE+XLkLE9saxozn
          private_networking: true
          monitoring: true
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
          kind: DigitalOceanProviderConfig
          apiVersion: digitaloceanproviderconfig/v1alpha1
          region: {{ .Provider.Location }}
          size: {{ .Provider.NodeSpec.SKU }}
          image: ubuntu-18-04-x64
          tags:
          - KubernetesCluster:{{ .Provider.ClusterName }}
          sshPublicKeys:
          - ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDa7/godjDMz8zAY0dMjujPQGoN/dUgH6b4E9WOUIEXQH9lbZx7yXRJ/COPHvXVUqVjNO8BkNHWGrDDr7Ozq8yfKz4xTFMRM9IsXFwopsC5ijd7XiHnwjJsDWAMbLje1AOL4MDvmSuVJUmr6ZuaAZUTg3zRredBxdiw0nj1pQEuHZ29DVmmoedM2CDxGMwR+sFOvgvkW4pJLUbq3uXxFN1z2t/djyO+YENHe3BRJ2jA9SMi+7KrN3Z3N09r6CtdeSRm/m3GDsreyWDJRsUJ9w1XGQc2qYcpDBycUPjBfD2nLeTxlZp5JAu74P+QTbmghoT9MudOqZE+XLkLE9saxozn
          private_networking: true
          monitoring: true
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
