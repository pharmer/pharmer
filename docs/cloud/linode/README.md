---
title: Linode Overview
menu:
product_pharmer_0.3.1
identifier: linode-overview
name: Overview
parent: linode
weight: 10
product_name: pharmer
menu_name: product_pharmer_0.3.1
section_menu_id: cloud
url: /products/pharmer/0.3.1/cloud/linode/
aliases:
- /products/pharmer/0.3.1/cloud/linode/README/
---

# Running Kubernetes on [Linode](https://linode.com)

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


### Cluster provisioning

There are two steps to create a Kubernetes cluster using `pharmer`. In first step `pharmer` create basic configuration file with user choice. Then in second step `pharmer` applies those information to create cluster on specific provider.

Here, we discuss how to use `pharmer` to create a Kubernetes cluster on `linode`

#### Cluster Creating

We want to create a cluster with following information:

- Provider: linode
- Cluster name: linode
- Location: us-central
- Number of master nodes: 3
- Number of worker nodes: 1
- Worker Node sku: g6-standard-2 (cpu: 2, memory: 7.5 Gb)
- Kubernetes version: v1.13.5
- Credential name: [linode](#credential-importing)

For location code and sku details click [hrere](https://github.com/pharmer/cloud/blob/master/data/json/apis/cloud.pharmer.io/v1/cloudproviders/linode.json)

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
$ pharmer create cluster linode-1 \
    --masters 3 \
    --provider linode \
    --zone us-central \
    --nodes g6-standard-2=1 \
    --credential-uid linode \
    --kubernetes-version v1.13.5
```

To know about [pod networks](https://kubernetes.io/docs/concepts/cluster-administration/networking/) supports in `pharmer` click [here](/docs/networking.md)

The directory structure of the storage provider will be look like:


```console
~/.pharmer/store.d/$USER/clusters/
├── l1
│   ├── machine
│   │   ├── l1-master-0.json
│   │   ├── l1-master-1.json
│   │   └── l1-master-2.json
│   ├── machineset
│   │   └── g6-standard-2-pool.json
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
│       ├── id_l1-sshkey
│       └── id_l1-sshkey.pub
└── l1.json

6 directories, 15 files
```


Here,
  - `machine`: conntains information about the master machines to be deployed
  - `machineset`: contains information about the machinesets to be deployed
  - `pki`: contains the cluster certificate information containing `ca`, `front-proxy-ca`, `etcd/ca` and service account keys `sa`
  - `ssh`: has the ssh credentials on cluster's nodes. With this key you can `ssh` into any node on a cluster
  - `l1.json`: contains the cluster resource information

You can view your cluster configuration file by following command.


```yaml
$ pharmer get cluster l1 -o yaml
kind: Cluster
apiVersion: cluster.pharmer.io/v1beta1
metadata:
  name: l1
  uid: 47ed5e2a-7856-11e9-8051-e0d55ee85d14
  generation: 1558064758076986000
  creationTimestamp: '2019-05-17T03:45:58Z'
spec:
  clusterApi:
    kind: Cluster
    apiVersion: cluster.k8s.io/v1alpha1
    metadata:
      name: l1
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
      region: us-central
      zone: us-central
      instanceImage: linode/ubuntu16.04lts
      networkProvider: calico
      ccmCredentialName: linode
      sshKeyName: l1-sshkey
      linode:
        rootPassword: 9GPOgQZbSZ4gwxT0
    kubernetesVersion: v1.13.5
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


You can modify this configuration by:
```console
$ pharmer edit cluster l1
```

#### Applying 

If everything looks ok, we can now apply the resources. This actually creates resources on `linode`.
Up to now we've only been working locally.

To apply run:

 ```console
$ pharmer apply l1
```

Now, `pharmer` will apply that configuration, this create a Kubernetes cluster. After completing task the configuration file of the cluster will be look like


```yaml
pharmer get cluster l1 -o yaml
---
kind: Cluster
apiVersion: cluster.pharmer.io/v1beta1
metadata:
  name: l1
  uid: 47ed5e2a-7856-11e9-8051-e0d55ee85d14
  generation: 1558065344523630600
  creationTimestamp: '2019-05-17T03:45:58Z'
spec:
  clusterApi:
    kind: Cluster
    apiVersion: cluster.k8s.io/v1alpha1
    metadata:
      name: l1
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
            label: l1-lb
            region: us-central
            tags: []
  config:
    masterCount: 3
    cloud:
      cloudProvider: linode
      region: us-central
      zone: us-central
      instanceImage: linode/ubuntu16.04lts
      networkProvider: calico
      ccmCredentialName: linode
      sshKeyName: l1-sshkey
      linode:
        rootPassword: 9GPOgQZbSZ4gwxT0
        kernelId: linode/latest-64bit
    kubernetesVersion: v1.13.5
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


Here,

  - `status.phase`: is ready. So, you can use your cluster from local machine.
  - `status.clusterApi.status.apiEndpoints` is the cluster's apiserver address

To get the `kubectl` configuration file(kubeconfig) on your local filesystem run the following command.

```console
$ pharmer use cluster l1
```
If you don't have `kubectl` installed click [here](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

Now you can run `kubectl get nodes` and verify that your kubernetes v1.13.5 is running.


Now you can run `kubectl get nodes` and verify that your kubernetes 1.13.5 is running.
```console
$ kubectl get nodes
NAME                       STATUS   ROLES    AGE     VERSION
l1-master-0                Ready    master   6m21s   v1.13.5
l1-master-1                Ready    master   3m10s   v1.13.5
l1-master-2                Ready    master   2m7s    v1.13.5
g6-standard-2-pool-5pft6   Ready    node     56s     v1.13.5
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
NAME                       AGE
g6-standard-2-pool-pft6v   4m
l1-master-0                4m
l1-master-1                4m
l1-master-2                4m

$ kubectl get machinesets
NAME                 AGE
g6-standard-2-pool   5m
```



#### Deploy new master machines
You can create new master machine by the deploying the following yaml

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
      region: us-central
      type: g6-standard-2
      image: linode/ubuntu16.04lts
      pubkey: ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC+e0+nNjre2JoHDySl1HiWsWmwLyIs1+KbQIM/15JKgQLA4PpkrVwOQkxjZOA+ozBoBgilAFLbca5ERUXeMz8nMoF1+JxwUoqSAeeA7PB1ZyU78r7c0nb1OMcMxituHZnpwtXxbsAAWMXBiEQx8IrlttJrlaD/7IP+jlSzwCyLOAVU/86euC+2bVi6SxnbVNy+POmrwncAx1VXHLP2o1zM9L+ENhYwR1YNfBeQse1zqSafgxEFU9SCrJGbnq6mJNS3U/dVg96Aj4QCYuC8wB7Nmca7U7/3EjTa+rXHzc5g1lqcHI2s26niK34kPLiu8vxp9k2Gkw2Me88Z4J60dScD
  versions:
    kubelet: v1.13.5
    controlPlane: v1.13.5
```

 

#### Create new worker machines

You can create new worker machines by deploying the following yaml


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
      region: us-central
      type: g6-standard-2
      image: linode/ubuntu16.04lts
      pubkey: ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC+e0+nNjre2JoHDySl1HiWsWmwLyIs1+KbQIM/15JKgQLA4PpkrVwOQkxjZOA+ozBoBgilAFLbca5ERUXeMz8nMoF1+JxwUoqSAeeA7PB1ZyU78r7c0nb1OMcMxituHZnpwtXxbsAAWMXBiEQx8IrlttJrlaD/7IP+jlSzwCyLOAVU/86euC+2bVi6SxnbVNy+POmrwncAx1VXHLP2o1zM9L+ENhYwR1YNfBeQse1zqSafgxEFU9SCrJGbnq6mJNS3U/dVg96Aj4QCYuC8wB7Nmca7U7/3EjTa+rXHzc5g1lqcHI2s26niK34kPLiu8vxp9k2Gkw2Me88Z4J60dScD
  versions:
    kubelet: v1.13.5
```


#### Create new machinesets

You can create new machinesets by deploying the following yaml


```yaml
kind: MachineSet
apiVersion: cluster.k8s.io/v1alpha1
metadata:
  name: g6-standard-2-pool
spec:
  replicas: 1
  selector:
    matchLabels:
      cluster.k8s.io/cluster-name: l1
      cluster.pharmer.io/mg: g6-standard-2
  template:
    metadata:
      labels:
        cluster.k8s.io/cluster-name: l1
        cluster.pharmer.io/cluster: l1
        cluster.pharmer.io/mg: g6-standard-2
        node-role.kubernetes.io/node: ''
        set: node
    spec:
      providerSpec:
        value:
          kind: LinodeClusterProviderConfig
          apiVersion: linodeproviderconfig/v1alpha1
          roles:
          - Node
          region: us-central
          type: g6-standard-2
          image: linode/ubuntu16.04lts
          pubkey: ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC+e0+nNjre2JoHDySl1HiWsWmwLyIs1+KbQIM/15JKgQLA4PpkrVwOQkxjZOA+ozBoBgilAFLbca5ERUXeMz8nMoF1+JxwUoqSAeeA7PB1ZyU78r7c0nb1OMcMxituHZnpwtXxbsAAWMXBiEQx8IrlttJrlaD/7IP+jlSzwCyLOAVU/86euC+2bVi6SxnbVNy+POmrwncAx1VXHLP2o1zM9L+ENhYwR1YNfBeQse1zqSafgxEFU9SCrJGbnq6mJNS3U/dVg96Aj4QCYuC8wB7Nmca7U7/3EjTa+rXHzc5g1lqcHI2s26niK34kPLiu8vxp9k2Gkw2Me88Z4J60dScD
      versions:
        kubelet: v1.13.5
```


#### Create new machine-deployments

You can create new machine-deployments by deploying the following yaml


```yaml
kind: MachineDeployment
apiVersion: cluster.k8s.io/v1alpha1
metadata:
  name: g6-standard-2-pool
spec:
  replicas: 1
  selector:
    matchLabels:
      cluster.k8s.io/cluster-name: l1
      cluster.pharmer.io/mg: g6-standard-2
  template:
    metadata:
      labels:
        cluster.k8s.io/cluster-name: l1
        cluster.pharmer.io/cluster: l1
        cluster.pharmer.io/mg: g6-standard-2
        node-role.kubernetes.io/node: ''
        set: node
    spec:
      providerSpec:
        value:
          kind: LinodeClusterProviderConfig
          apiVersion: linodeproviderconfig/v1alpha1 
          roles:
          - Node
          region: us-central
          type: g6-standard-2
          image: linode/ubuntu16.04lts
          pubkey: ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC+e0+nNjre2JoHDySl1HiWsWmwLyIs1+KbQIM/15JKgQLA4PpkrVwOQkxjZOA+ozBoBgilAFLbca5ERUXeMz8nMoF1+JxwUoqSAeeA7PB1ZyU78r7c0nb1OMcMxituHZnpwtXxbsAAWMXBiEQx8IrlttJrlaD/7IP+jlSzwCyLOAVU/86euC+2bVi6SxnbVNy+POmrwncAx1VXHLP2o1zM9L+ENhYwR1YNfBeQse1zqSafgxEFU9SCrJGbnq6mJNS3U/dVg96Aj4QCYuC8wB7Nmca7U7/3EjTa+rXHzc5g1lqcHI2s26niK34kPLiu8vxp9k2Gkw2Me88Z4J60dScD
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
$ pharmer delete cluster l1
```

Then, the yaml file looks like


```yaml
$ pharmer get cluster l1 -o yaml
kind: Cluster
apiVersion: cluster.pharmer.io/v1beta1
metadata:
  name: l1
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


Here,

- `metadata.deletionTimestamp`: is set when cluster deletion command was applied.

Now, to apply delete operation of the cluster, run

```console
$ pharmer apply l1
```

**Congratulations !!!** , you're an official `pharmer` user now.

