---
title: DigitalOcean Overview
menu:
product_pharmer_0.3.0
identifier: digitalocean-overview
name: Overview
parent: digitalocean
weight: 10
product_name: pharmer
menu_name: product_pharmer_0.3.0
section_menu_id: cloud
url: /products/pharmer/0.3.0/cloud/digitalocean/
aliases:
- /products/pharmer/0.3.0/cloud/digitalocean/README/
---

# Running Kubernetes on [DigitalOcean](https://cloud.digitalocean.com)

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



### Cluster provisioning

There are two steps to create a Kubernetes cluster using `pharmer`. In first step `pharmer` create basic configuration file with user choice. Then in second step `pharmer` applies those information to create cluster on specific provider.

Here, we discuss how to use `pharmer` to create a Kubernetes cluster on `digitalocean`

#### Cluster Creating

We want to create a cluster with following information:

- Provider: digitalocean
- Cluster name: digitalocean
- Location: nyc1
- Number of master nodes: 3
- Number of worker nodes: 1
- Worker Node sku: 2gb (cpu: 1, memory: 2 Gb)
- Kubernetes version: v1.13.5
- Credential name: [digitalocean](#credential-importing)

For location code and sku details click [hrere](https://github.com/pharmer/cloud/blob/master/data/json/apis/cloud.pharmer.io/v1/cloudproviders/digitalocean.json)

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
$ pharmer create cluster digitalocean-1 \
    --masters 3 \
    --provider digitalocean \
    --zone nyc1 \
    --nodes 2gb=1 \
    --credential-uid digitalocean \
    --kubernetes-version v1.13.5
```

To know about [pod networks](https://kubernetes.io/docs/concepts/cluster-administration/networking/) supports in `pharmer` click [here](/docs/networking.md)

The directory structure of the storage provider will be look like:



```console
$ tree ~/.pharmer/store.d/$USER/clusters/
/home/<user>/.pharmer/store.d/<user>/clusters/
├── d1
│   ├── machine
│   │   ├── d1-master-0.json
│   │   ├── d1-master-1.json
│   │   └── d1-master-2.json
│   ├── machineset
│   │   └── 2gb-pool.json
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
│       ├── id_d1-sshkey
│       └── id_d1-sshkey.pub
└── d1.json

6 directories, 15 files
```


Here,
  - `machine`: conntains information about the master machines to be deployed
  - `machineset`: contains information about the machinesets to be deployed
  - `pki`: contains the cluster certificate information containing `ca`, `front-proxy-ca`, `etcd/ca` and service account keys `sa`
  - `ssh`: has the ssh credentials on cluster's nodes. With this key you can `ssh` into any node on a cluster
  - `d1.json`: contains the cluster resource information

You can view your cluster configuration file by following command.



```yaml
$ pharmer get cluster d1 -o yaml
kind: Cluster
apiVersion: cluster.pharmer.io/v1beta1
metadata:
  name: d1
  uid: 379d4d7f-77c1-11e9-b997-e0d55ee85d14
  generation: 1558000735696016100
  creationTimestamp: '2019-05-16T09:58:55Z'
spec:
  clusterApi:
    kind: Cluster
    apiVersion: cluster.k8s.io/v1alpha1
    metadata:
      name: d1
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
    3
    cloud:
      cloudProvider: digitalocean
      region: nyc1
      zone: nyc1
      instanceImage: ubuntu-18-04-x64
      networkProvider: calico
      sshKeyName: d1-sshkey
    kubernetesVersion: v1.13.5
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


You can modify this configuration by:
```console
$ pharmer edit cluster d1
```

#### Applying 

If everything looks ok, we can now apply the resources. This actually creates resources on `digitalocean`.
Up to now we've only been working locally.

To apply run:

 ```console
$ pharmer apply d1
```

Now, `pharmer` will apply that configuration, this create a Kubernetes cluster. After completing task the configuration file of the cluster will be look like


```yaml
$ pharmer get cluster d1 -o yaml
kind: Cluster
apiVersion: cluster.pharmer.io/v1beta1
metadata:
  name: d1
  uid: 379d4d7f-77c1-11e9-b997-e0d55ee85d14
  generation: 1558000735696016100
  creationTimestamp: '2019-05-16T09:58:55Z'
spec:
  clusterApi:
    kind: Cluster
    apiVersion: cluster.k8s.io/v1alpha1
    metadata:
      name: d1
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
          name: d1-lb
          region: nyc1
          status: active
          sticky_sessions:
            type: none
        metadata:
          creationTimestamp:
  config:
    3
    cloud:
      cloudProvider: digitalocean
      region: nyc1
      zone: nyc1
      instanceImage: ubuntu-18-04-x64
      networkProvider: calico
      sshKeyName: d1-sshkey
    kubernetesVersion: v1.13.5
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


Here,

  - `status.phase`: is ready. So, you can use your cluster from local machine.
  - `status.clusterApi.status.apiEndpoints` is the cluster's apiserver address

To get the `kubectl` configuration file(kubeconfig) on your local filesystem run the following command.

```console
$ pharmer use cluster d1
```
If you don't have `kubectl` installed click [here](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

Now you can run `kubectl get nodes` and verify that your kubernetes v1.13.5 is running.


```console
$ kubectl get nodes

NAME             STATUS   ROLES    AGE   VERSION
2gb-pool-p2c7m   Ready    node     13m   v1.13.5
d1-master-0      Ready    master   29m   v1.13.5
d1-master-1      Ready    master   14m   v1.13.5
d1-master-2      Ready    master   13m   v1.13.5
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
NAME             AGE
2gb-pool-p2c7m   1m
d1-master-0      2m
d1-master-1      2m
d1-master-2      2m

$ kubectl get machinesets
NAME       AGE
2gb-pool   2m
```



#### Deploy new master machines
You can create new master machine by the deploying the following yaml

```yaml
kind: Machine
apiVersion: cluster.k8s.io/v1alpha1
metadata:
  name: d1-master-3
  creationTimestamp: '2019-05-16T09:58:56Z'
  labels:
    cluster.k8s.io/cluster-name: d1
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
      region: nyc1
      size: 2gb
      image: ubuntu-18-04-x64
      tags:
      - KubernetesCluster:d1
      sshPublicKeys:
      - ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDa7/godjDMz8zAY0dMjujPQGoN/dUgH6b4E9WOUIEXQH9lbZx7yXRJ/COPHvXVUqVjNO8BkNHWGrDDr7Ozq8yfKz4xTFMRM9IsXFwopsC5ijd7XiHnwjJsDWAMbLje1AOL4MDvmSuVJUmr6ZuaAZUTg3zRredBxdiw0nj1pQEuHZ29DVmmoedM2CDxGMwR+sFOvgvkW4pJLUbq3uXxFN1z2t/djyO+YENHe3BRJ2jA9SMi+7KrN3Z3N09r6CtdeSRm/m3GDsreyWDJRsUJ9w1XGQc2qYcpDBycUPjBfD2nLeTxlZp5JAu74P+QTbmghoT9MudOqZE+XLkLE9saxozn
      private_networking: true
      monitoring: true
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
  name: worker-1
  labels:
    cluster.k8s.io/cluster-name: d1
    node-role.kubernetes.io/master: ''
    set: node
spec:
  metadata:
    creationTimestamp:
  providerSpec:
    value:
      kind: DigitalOceanProviderConfig
      apiVersion: digitaloceanproviderconfig/v1alpha1
      region: nyc1
      size: 2gb
      image: ubuntu-18-04-x64
      tags:
      - KubernetesCluster:d1
      sshPublicKeys:
      - ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDa7/godjDMz8zAY0dMjujPQGoN/dUgH6b4E9WOUIEXQH9lbZx7yXRJ/COPHvXVUqVjNO8BkNHWGrDDr7Ozq8yfKz4xTFMRM9IsXFwopsC5ijd7XiHnwjJsDWAMbLje1AOL4MDvmSuVJUmr6ZuaAZUTg3zRredBxdiw0nj1pQEuHZ29DVmmoedM2CDxGMwR+sFOvgvkW4pJLUbq3uXxFN1z2t/djyO+YENHe3BRJ2jA9SMi+7KrN3Z3N09r6CtdeSRm/m3GDsreyWDJRsUJ9w1XGQc2qYcpDBycUPjBfD2nLeTxlZp5JAu74P+QTbmghoT9MudOqZE+XLkLE9saxozn
      private_networking: true
      monitoring: true
  versions:
    kubelet: v1.13.5
```


#### Create new machinesets

You can create new machinesets by deploying the following yaml


```yaml
kind: MachineSet
apiVersion: cluster.k8s.io/v1alpha1
metadata:
  name: 2gb-pool
spec:
  replicas: 1
  selector:
    matchLabels:
      cluster.k8s.io/cluster-name: d1
      cluster.pharmer.io/mg: 2gb
  template:
    metadata:
      labels:
        cluster.k8s.io/cluster-name: d1
        cluster.pharmer.io/cluster: d1
        cluster.pharmer.io/mg: 2gb
        node-role.kubernetes.io/node: ''
        set: node
    spec:
      providerSpec:
        value:
          kind: DigitalOceanProviderConfig
          apiVersion: digitaloceanproviderconfig/v1alpha1
          region: nyc1
          size: 2gb
          image: ubuntu-18-04-x64
          tags:
          - KubernetesCluster:d1
          sshPublicKeys:
          - ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDa7/godjDMz8zAY0dMjujPQGoN/dUgH6b4E9WOUIEXQH9lbZx7yXRJ/COPHvXVUqVjNO8BkNHWGrDDr7Ozq8yfKz4xTFMRM9IsXFwopsC5ijd7XiHnwjJsDWAMbLje1AOL4MDvmSuVJUmr6ZuaAZUTg3zRredBxdiw0nj1pQEuHZ29DVmmoedM2CDxGMwR+sFOvgvkW4pJLUbq3uXxFN1z2t/djyO+YENHe3BRJ2jA9SMi+7KrN3Z3N09r6CtdeSRm/m3GDsreyWDJRsUJ9w1XGQc2qYcpDBycUPjBfD2nLeTxlZp5JAu74P+QTbmghoT9MudOqZE+XLkLE9saxozn
          private_networking: true
          monitoring: true
      versions:
        kubelet: v1.13.5
```


#### Create new machine-deployments

You can create new machine-deployments by deploying the following yaml


```yaml
kind: MachineDeployment
apiVersion: cluster.k8s.io/v1alpha1
metadata:
  name: 2gb-pool
spec:
  replicas: 1
  selector:
    matchLabels:
      cluster.k8s.io/cluster-name: d1
      cluster.pharmer.io/mg: 2gb
  template:
    metadata:
      labels:
        cluster.k8s.io/cluster-name: d1
        cluster.pharmer.io/cluster: d1
        cluster.pharmer.io/mg: 2gb
        node-role.kubernetes.io/node: ''
        set: node
    spec:
      providerSpec:
        value:
          kind: DigitalOceanProviderConfig
          apiVersion: digitaloceanproviderconfig/v1alpha1
          region: nyc1
          size: 2gb
          image: ubuntu-18-04-x64
          tags:
          - KubernetesCluster:d1
          sshPublicKeys:
          - ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDa7/godjDMz8zAY0dMjujPQGoN/dUgH6b4E9WOUIEXQH9lbZx7yXRJ/COPHvXVUqVjNO8BkNHWGrDDr7Ozq8yfKz4xTFMRM9IsXFwopsC5ijd7XiHnwjJsDWAMbLje1AOL4MDvmSuVJUmr6ZuaAZUTg3zRredBxdiw0nj1pQEuHZ29DVmmoedM2CDxGMwR+sFOvgvkW4pJLUbq3uXxFN1z2t/djyO+YENHe3BRJ2jA9SMi+7KrN3Z3N09r6CtdeSRm/m3GDsreyWDJRsUJ9w1XGQc2qYcpDBycUPjBfD2nLeTxlZp5JAu74P+QTbmghoT9MudOqZE+XLkLE9saxozn
          private_networking: true
          monitoring: true
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
$ pharmer delete cluster d1
```

Then, the yaml file looks like


```yaml
$ pharmer get cluster d1 -o yaml
kind: Cluster
apiVersion: cluster.pharmer.io/v1beta1
metadata:
  name: d1
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
$ pharmer apply d1
```

**Congratulations !!!** , you're an official `pharmer` user now.

