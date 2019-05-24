---
title: Packet Overview
menu:
product_pharmer_0.3.1
identifier: packet-overview
name: Overview
parent: packet
weight: 10
product_name: pharmer
menu_name: product_pharmer_0.3.1
section_menu_id: cloud
url: /products/pharmer/0.3.1/cloud/packet/
aliases:
- /products/pharmer/0.3.1/cloud/packet/README/
---

# Running Kubernetes on [Packet](https://app.packet.net)

Following example will use `pharmer` to create a Kubernetes cluster with 1 worker nodes and 3 master nodes (i,e, 4 nodes in you cluster).

### Before you start

As a prerequisite, you need to have `pharmer` installed.  To install `pharmer` run the following command.

```console
$ mkdir -p $(go env GOPATH)/src/github.com/pharmer
$ cd $(go env GOPATH)/src/github.com/pharmer
$ git clone https://github.com/pharmer/pharmer
$ cd pharmer
$ ./hack/make.py

$ pharmer -h
```

### Pharmer storage

To store your cluster  and credential resource, `pharmer` use [vfs](/docs/cli/vfs.md) as default storage provider. There is another provider [postgres database](/docs/cli/xorm.md) available for storing resources.

To know more click [here](/docs/cli/datastore.md)

In this document we will use local file system ([vfs](/docs/cli/vfs.md)) as a storage provider.

### Credential importing


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


### Cluster provisioning

There are two steps to create a Kubernetes cluster using `pharmer`. In first step `pharmer` create basic configuration file with user choice. Then in second step `pharmer` applies those information to create cluster on specific provider.

Here, we discuss how to use `pharmer` to create a Kubernetes cluster on `packet`

#### Cluster Creating

We want to create a cluster with following information:

- Provider: packet
- Cluster name: packet
- Location: ewr1
- Number of master nodes: 1
- Number of worker nodes: 1
- Worker Node sku: baremetal_0 (cpu: 4 x86 64bit, memory: 8GB DDR3)
- Kubernetes version: v1.13.5
- Credential name: [packet](#credential-importing)

For location code and sku details click [hrere](https://github.com/pharmer/cloud/blob/master/data/json/apis/cloud.pharmer.io/v1/cloudproviders/packet.json)

Available options in `pharmer` to create a cluster are:
 ```console
 $ pharmer create cluster -h
 Create a Kubernetes cluster for a given cloud provider

Usage:
  pharmer create cluster [flags]

Aliases:
  cluster, clusters, Cluster

Examples:
pharmer create cluster demo-cluster

Flags:
      --credential-uid string       Use preconfigured cloud credential uid
  -h, --help                        help for cluster
      --kubernetes-version string   Kubernetes version
      --masters int                 Number of masters (default 1)
      --namespace string            Namespace (default "default")
      --network-provider string     Name of CNI plugin. Available options: calico, flannel, kubenet, weavenet (default "calico")
      --nodes stringToInt           Node set configuration (default [])
  -o, --owner string                Current user id (default "tahsin")
      --provider string             Provider name
      --zone string                 Cloud provider zone name

Global Flags:
      --alsologtostderr                  log to standard error as well as files
      --analytics                        Send analytical events to Google Guard (default true)
      --config-file string               Path to Pharmer config file
      --env string                       Environment used to enable debugging (default "prod")
      --kubeconfig string                Paths to a kubeconfig. Only required if out-of-cluster.
      --log_backtrace_at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                   If non-empty, write log files in this directory
      --logtostderr                      log to standard error instead of files (default true)
      --master string                    The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.
      --stderrthreshold severity         logs at or above this threshold go to stderr
  -v, --v Level                          log level for V logs
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging
 ```

So, we need to run following command to create cluster with our information.

```console
$ pharmer create cluster packet-1 \
    --masters 1 \
    --provider packet \
    --zone ewr1 \
    --nodes baremetal_0=1 \
    --credential-uid packet \
    --kubernetes-version v1.13.5
```

To know about [pod networks](https://kubernetes.io/docs/concepts/cluster-administration/networking/) supports in `pharmer` click [here](/docs/networking.md)

The directory structure of the storage provider will be look like:


```console
~/.pharmer/store.d/$USER/clusters/
/home/<user>/.pharmer/store.d/<user>/clusters/
├── p1
│   ├── machine
│   │   ├── p1-master-0.json
│   ├── machineset
│   │   └── baremetal-0-pool.json
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
│       ├── id_p1-sshkey
│       └── id_p1-sshkey.pub
└── p1.json

6 directories, 13 files
```


Here,
  - `machine`: conntains information about the master machines to be deployed
  - `machineset`: contains information about the machinesets to be deployed
  - `pki`: contains the cluster certificate information containing `ca`, `front-proxy-ca`, `etcd/ca` and service account keys `sa`
  - `ssh`: has the ssh credentials on cluster's nodes. With this key you can `ssh` into any node on a cluster
  - `p1.json`: contains the cluster resource information

You can view your cluster configuration file by following command.


```yaml
$ pharmer get cluster p1 -o yaml
kind: Cluster
apiVersion: cluster.pharmer.io/v1beta1
metadata:
  name: p1
  uid: a057bb8d-785a-11e9-901f-e0d55ee85d14
  generation: 1558066624400477400
  creationTimestamp: '2019-05-17T04:17:04Z'
spec:
  clusterApi:
    kind: Cluster
    apiVersion: cluster.k8s.io/v1alpha1
    metadata:
      name: p1
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
      region: ewr1
      zone: ewr1
      instanceImage: ubuntu_16_04
      networkProvider: calico
      ccmCredentialName: pack
      sshKeyName: p1-sshkey
    kubernetesVersion: v1.13.5
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


You can modify this configuration by:
```console
$ pharmer edit cluster p1
```

#### Applying 

If everything looks ok, we can now apply the resources. This actually creates resources on `packet`.
Up to now we've only been working locally.

To apply run:

 ```console
$ pharmer apply p1
```

Now, `pharmer` will apply that configuration, this create a Kubernetes cluster. After completing task the configuration file of the cluster will be look like


```yaml
$ pharmer get cluster p1 -o yaml
kind: Cluster
apiVersion: cluster.pharmer.io/v1beta1
metadata:
  name: p1
  uid: 157599fa-7861-11e9-9009-e0d55ee85d14
  generation: 1558069397870031400
  creationTimestamp: '2019-05-17T05:03:17Z'
spec:
  clusterApi:
    kind: Cluster
    apiVersion: cluster.k8s.io/v1alpha1
    metadata:
      name: p1
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
      region: ewr1
      zone: ewr1
      instanceImage: ubuntu_16_04
      networkProvider: calico
      ccmCredentialName: pack
      sshKeyName: p1-sshkey
    kubernetesVersion: v1.13.5
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


Here,

  - `status.phase`: is ready. So, you can use your cluster from local machine.
  - `status.clusterApi.status.apiEndpoints` is the cluster's apiserver address

To get the `kubectl` configuration file(kubeconfig) on your local filesystem run the following command.

```console
$ pharmer use cluster p1
```
If you don't have `kubectl` installed click [here](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

Now you can run `kubectl get nodes` and verify that your kubernetes v1.13.5 is running.


```console
$ kubectl get nodes
p1-master-0        Ready    master   29m   v1.13.5
baremetal-0-pool   Ready    node     13m   v1.13.5
```





### Cluster Scaling

Scaling a cluster refers following meanings
- Add new master and worker machines
- Increment the number of nodes of a certain machine-set and machine-deployment
- Decrement the number of nodes of a certain machine-set and machine-deployment
- Introduce a new machine-set and machine-deployment with a number of nodes
- Delete existing machine, machine-set and machine-deployments

You can see the machine and machine-sets deployed in the cluster


```console
$ kubectl get machines
NAME               AGE
baremetal-0-pool   1m
p1-master-0        2m

$ kubectl get machinesets
NAME               AGE
baremetal-0-pool   2m
```


 

#### Create new worker machines

You can create new worker machines by deploying the following yaml


```yaml
kind: Machine
apiVersion: cluster.k8s.io/v1alpha1
metadata:
  name: worker-1
  labels:
    cluster.k8s.io/cluster-name: p1
    node-role.kubernetes.io/master: ''
    set: node
spec:
  providerSpec:
    value:
      kind: PacketClusterProviderConfig
      apiVersion: Packetproviderconfig/v1alpha1
      plan: baremetal_0
      type: Regular
  versions:
    kubelet: v1.13.5
```


#### Create new machinesets

You can create new machinesets by deploying the following yaml


```yaml
kind: MachineSet
apiVersion: cluster.k8s.io/v1alpha1
metadata:
  name: baremetal-0-pool
spec:
  replicas: 1
  selector:
    matchLabels:
      cluster.k8s.io/cluster-name: p1
      cluster.pharmer.io/mg: baremetal_0
  template:
    metadata:
      labels:
        cluster.k8s.io/cluster-name: p1
        cluster.pharmer.io/cluster: p1
        cluster.pharmer.io/mg: baremetal_0
        node-role.kubernetes.io/node: ''
        set: node
    spec:
      providerSpec:
        value:
          kind: PacketClusterProviderConfig
          apiVersion: Packetproviderconfig/v1alpha1
          plan: baremetal_0
          type: Regular
      versions:
        kubelet: v1.13.5
```


#### Create new machine-deployments

You can create new machine-deployments by deploying the following yaml


```yaml
kind: MachineDeployment
apiVersion: cluster.k8s.io/v1alpha1
metadata:
  name: baremetal-0-pool
spec:
  replicas: 1
  selector:
    matchLabels:
      cluster.k8s.io/cluster-name: p1
      cluster.pharmer.io/mg: baremetal_0
  template:
    metadata:
      labels:
        cluster.k8s.io/cluster-name: p1
        cluster.pharmer.io/cluster: p1
        cluster.pharmer.io/mg: baremetal_0
        node-role.kubernetes.io/node: ''
        set: node
    spec:
      providerSpec:
        value:
          kind: PacketClusterProviderConfig
          apiVersion: Packetproviderconfig/v1alpha1
          plan: baremetal_0
          type: Regular
      versions:
        kubelet: v1.13.5
```


#### Scale Cluster

You can also update number of nodes of an existing machine-set and machine-deployment using

```console
$ kubectl edit <machineset-name> 
$ kubectl edit <machinedeployment-name> 
```
and update the `spec.replicas` field

#### Delete nodes

You can delete machines using

```console
$ kubectl delete machine <machine-name>
```
Warning: if the machine is controlled by a machineset, a new machine will be created. You should update/delete machineset in that case

You can delete machine-set and machine-deployments using

```console
$ kubectl delete machineset <machineset-name>
$ kubectl delete machinedeployment <machinedeployment-name>
```

### Cluster Upgrading

#### Upgrade master machines

You can deploy new master machines with specifying new version in `spec.version.controlPlane` and `spec.version.kubelet`. After new master machines are ready, you can safely delete old ones

#### Upgrade worker machines

You can upgrade worker machines by editing machine-deployment

``` console
$ kubectl edit machinedeployments <machinedeployment-name>
```

and updating the `spec.version.kubelet`

To upgrade machinesets, you have to deploy new machinesets with specifying new version in `spec.template.spec.version.kubelet`
After new machines are ready, you can safely delete old machine-sets

## Cluster Deleting

To delete your cluster run

```console
$ pharmer delete cluster p1
```

Then, the yaml file looks like


```yaml
$ pharmer get cluster p1 -o yaml
kind: Cluster
apiVersion: cluster.pharmer.io/v1beta1
metadata:
  name: p1
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


Here,

- `metadata.deletionTimestamp`: is set when cluster deletion command was applied.

Now, to apply delete operation of the cluster, run

```console
$ pharmer apply p1
```

**Congratulations !!!** , you're an official `pharmer` user now.

