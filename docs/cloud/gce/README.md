---
title: GCE Overview
menu:
  product_pharmer_0.1.0-alpha.1:
    identifier: gce-overview
    name: Overview
    parent: gce
    weight: 10
product_name: pharmer
menu_name: product_pharmer_0.1.0-alpha.1
section_menu_id: cloud
url: /products/pharmer/0.1.0-alpha.1/cloud/gce/
aliases:
  - /products/pharmer/0.1.0-alpha.1/cloud/gce/README/
---

# Running Kubernetes on [Google Cloud Service](https://console.cloud.google.com)

Following example will use `pharmer ` to create a Kubernetes cluster with 1 worker node instance and a master instance (i,e, 2 instances in you cluster).

### Before you start

As a prerequisite, you need to have `pharmer` installed.  To install `pharmer` run the following command.

```console
mkdir -p $(go env GOPATH)/src/github.com/pharmer
cd $(go env GOPATH)/src/github.com/pharmer
git clone https://github.com/pharmer/pharmer
cd pharmer
go install -v

pharmer -h
```

### Pharmer storage

To store your cluster  and credential resource, `pharmer` use [vfs](/docs/cli/vfs.md) as default storage
provider. There is another provider [postgres database](/docs/cli/xorm.md) available for storing resources.

To know more click [here](/docs/cli/datastore.md)

In this document we will use local file system ([vfs](/docs/cli/vfs.md)) as a storage provider.

### Credential importing

 * **Issuing new credential**

You can issue a new credential for your `gce` project by running
```console
$ pharmer issue credential --provider=GoogleCloud gce
```

Here,
 - 'GoogleCloud' is cloud provider name
 - `gce` is credential name

Store the credential on a file and use that while importing credentials on pharmer.

From command line, run the following command

```console
$ pharmer create credential --from-file=<file-location> gce
```

Here, `gce` is the credential name, which must be unique within your storage.

To view credential file you can run:

```yaml
$ pharmer get credentials gce -o yaml
apiVersion: v1alpha1
kind: Credential
metadata:
  creationTimestamp: 2017-10-17T04:25:30Z
  name: gce
spec:
  data:
    projectID: k8s-qa
    serviceAccount: |
      {
        "type": "service_account",
        "project_id": "k8s-qa",
        "private_key_id": "private_key id",
        "private_key": "private_key",
        "client_email": "k8s-qa@k8s-qa.iam.gserviceaccount.com",
        "client_id": "client_id",
        "auth_uri": "https://accounts.google.com/o/oauth2/auth",
        "token_uri": "https://accounts.google.com/o/oauth2/token",
        "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
        "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/k8s-qa%40k8s-qa.iam.gserviceaccount.com"
      }
  provider: GoogleCloud

```
Here,
 - `spec.data.projectID` is the gce project id
 - `spec.data.serviceAccount` is the service account credential which can be edited by following command:
```console
$ phrmer edit credential gce
```
To see the all credentials you need to run following command.

```console
$ pharmer get credentials
NAME         Provider       Data
gce          GoogleCloud    projectID=k8s-qa, serviceAccount=<data>
```
You can also see the stored credential from the following location:
```console
~/.pharmer/store.d/credentials/gce.json
```

### Cluster provisioning

There are two steps to create a Kubernetes cluster using `pharmer`.
In first step `pharmer` create basic configuration file with user choice. Then in second step `pharmer` applies those
information to create cluster on specific provider.

Here, we discuss how to use `pharmer` to create a Kubernetes cluster on `gce`
 * **Cluster Creating:** We want to create a cluster with following information:
    - Provider: Google Cloud
    - Cluster name: g1
    - Location: us-central1-f (Central US)
    - Number of nodes: 1
    - Node sku: n1-standard-2 (cpu:2, ram: 7.5)
    - Kubernetes version: 1.11.0
    - Credential name: [gce](#credential-importing)

For location code and sku details click [hrere](https://github.com/pharmer/pharmer/blob/master/data/files/gce/cloud.json)

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
      --network-provider string     Name of CNI plugin. Available options: calico, flannel, kubenet, weavenet (default "calico")
      --nodes stringToInt           Node set configuration (default [])
      --provider string             Provider name
      --zone string                 Cloud provider zone name

Global Flags:
      --alsologtostderr                  log to standard error as well as files
      --analytics                        Send analytical events to Google Guard (default true)
      --config-file string               Path to Pharmer config file
      --env string                       Environment used to enable debugging (default "dev")
      --log_backtrace_at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                   If non-empty, write log files in this directory
      --logtostderr                      log to standard error instead of files (default true)
      --stderrthreshold severity         logs at or above this threshold go to stderr (default 2)
  -v, --v Level                          log level for V logs
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging
 ```

 So, we need to run following command to create cluster with our information.

```console
$ pharmer create cluster g1 \
	--v=5 \
	--provider=gce \
	--zone=us-central1-f \
	--nodes=n1-standard-2=1 \
	--credential-uid=gce \
	--kubernetes-version=v1.11.0
```

To know about [pod networks](https://kubernetes.io/docs/concepts/cluster-administration/networking/) supports in `pharmer` click [here](/docs/networking.md)

The directory structure of the storage provider will be look like:

```console
~/.pharmer/store.d/clusters/
        |-- v1
        |    |__ nodegroups
        |    |       |__ master.json
        |    |       |
        |    |       |__ n1-standard-2-pool.json
        |    |
        |    |--- pki
        |    |     |__ ca.crt
        |    |     |
        |    |     |__ ca.key
        |    |     |
        |    |     |__ front-proxy-ca.crt
        |    |     |
        |    |     |__ fron-proxy-ca.key
        |    |
        |    |__ ssh
        |          |__ id_g1-fwctge
        |          |
        |          |__ id_g1-fwctge.pub
        |
        |__ g1.json
```
Here,

   - `/v1/nodegroups/`: contains the node groups information. [Check below](#cluster-scaling) for node group operations.You can see the node group list using following command.
   ```console
$ pharmer get nodegroups -k g1
```
   - `v1/pki`: contains the cluster certificate information containing `ca` and `front-proxy-ca`.
   - `v1/ssh`: has the ssh credentials on cluster's nodes. With this key you can `ssh` into any node on a cluster
   - `v1.json`: contains the cluster resource information

You can view your cluster configuration file by following command.
```yaml
$ pharmer get cluster g1 -o yaml
apiVersion: v1alpha1
kind: Cluster
metadata:
  creationTimestamp: 2017-11-30T05:40:53Z
  generation: 1512020453910376439
  name: g1
  uid: 07fd49c7-d591-11e7-b0c0-382c4a73a7c4
spec:
  api:
    advertiseAddress: ""
    bindPort: 6443
  apiServerExtraArgs:
    cloud-config: /etc/kubernetes/ccm/cloud-config
    kubelet-preferred-address-types: Hostname,InternalDNS,InternalIP,ExternalDNS,ExternalIP
  caCertName: ca
  cloud:
    ccmCredentialName: gce
    cloudProvider: gce
    gce:
      NetworkName: default
      NodeTags:
      - g1-node
    instanceImage: ubuntu-1604-xenial-v20170721
    instanceImageProject: ubuntu-os-cloud
    region: us-central1
    sshKeyName: g1-fwctge
    zone: us-central1-f
  controllerManagerExtraArgs:
    cloud-config: /etc/kubernetes/ccm/cloud-config
  credentialName: gce
  frontProxyCACertName: front-proxy-ca
  kubernetesVersion: v1.11.0
  networking:
    networkProvider: calico
    nonMasqueradeCIDR: 10.0.0.0/8
    podSubnet: 10.244.0.0/16
status:
  cloud: {}
  phase: Pending
```
Here,

* `metadata.name` refers the cluster name, which should be unique within your cluster list.
* `metadata.uid` is a unique ACID, which is generated by pharmer
* `spec.cloud` specifies the cloud provider information. pharmer uses `ubuntu-16-04-x64` image by default. don't change the instance images, otherwise cluster may not be working.
* `spc.cloud.sshKeyName` shows which ssh key added to cluster instance.
* `spec.api.bindPort` is the api server port.
* `spec.networking` specifies the network information of the cluster
    * `networkProvider`: by default it is `calico`. To modify it click [here](/docs/networking.md).
    * `podSubnet`: in order for network policy to work correctly this field is needed. For flannel it will be `10.244.0.0/16`
* `spec.kubernetesVersion` is the cluster server version. It can be modified.
* `spec.credentialName` is the credential name which is provider during cluster creation command.
* `spec.apiServerExtraArgs` specifies which value will be forwarded to apiserver during cluster installation.
* `spec.authorizationMode` refers the cluster authorization mode
* `status.phase` may be `Pending`, `Ready`, `Deleting`, `Deleted`, `Upgrading` depending on current cluster status.

You can modify this configuration by:
```console
$ pharmer edit cluster g1
```

* **Applying:** If everything looks ok, we can now apply the resources. This actually creates resources on `GCE`.
 Up to now we've only been working locally.

 To apply run:
 ```console
$ pharmer apply g1
```

 Now, `pharmer` will apply that configuration, thus create a Kubernetes cluster. After completing task the configuration file of
 the cluster will be look like
```yaml
 $ pharmer get cluster g1 -o yaml
apiVersion: v1alpha1
kind: Cluster
metadata:
  creationTimestamp: 2017-11-30T05:40:53Z
  generation: 1512020453910376439
  name: g1
  uid: 07fd49c7-d591-11e7-b0c0-382c4a73a7c4
spec:
  api:
    advertiseAddress: ""
    bindPort: 6443
  apiServerExtraArgs:
    cloud-config: /etc/kubernetes/ccm/cloud-config
    kubelet-preferred-address-types: Hostname,InternalDNS,InternalIP,ExternalDNS,ExternalIP
  caCertName: ca
  cloud:
    ccmCredentialName: gce
    cloudProvider: gce
    gce:
      NetworkName: default
      NodeTags:
      - g1-node
    instanceImage: ubuntu-1604-xenial-v20170721
    instanceImageProject: ubuntu-os-cloud
    project: k8s-qa
    region: us-central1
    sshKeyName: g1-fwctge
    zone: us-central1-f
  controllerManagerExtraArgs:
    cloud-config: /etc/kubernetes/ccm/cloud-config
  credentialName: gce
  frontProxyCACertName: front-proxy-ca
  kubernetesVersion: v1.11.0
  networking:
    networkProvider: calico
    nonMasqueradeCIDR: 10.0.0.0/8
    podSubnet: 10.244.0.0/16
status:
  apiServer:
  - address: 10.128.0.2
    type: InternalIP
  - address: 104.154.163.23
    type: ExternalIP
  cloud: {}
  phase: Ready
```
Here,

  - `status.phase`: is ready. So, you can use your cluster from local machine.
  - `status.apiserver` is the cluster's apiserver address
  - `status.cloud.gce` contains provider resource information that are created by `pharmer` while creating cluster.


To get the `kubectl` configuration file(kubeconfig) on your local filesystem run the following command.
```console
$ pharmer use cluster g1
```
If you don't have `kubectl` installed click [here](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

Now you can run `kubectl get nodes` and verify that your kubernetes 1.11.0 is running.
```console
$ kubectl get nodes
NAME                             STATUS    ROLES     AGE       VERSION
g1-master                        Ready     master    4m        v1.11.0
n1-standard-2-pool-hdpldt-qnc2   Ready     node      57s       v1.11.0
```
If you want to `ssh` into your instance run the following command
```console
$ pharmer ssh node g1-master -k g1
```

### Cluster Scaling

Scaling a cluster refers following meanings:-
 1. Increment the number of nodes of a certain node group
 2. Decrement the number of nodes of a certain node group
 3. Introduce a new node group with a number of nodes
 4. Drop existing node group

To see the current node groups list, you need to run following command:
```console
$ pharmer get nodegroup -k g1
NAME                 Cluster   Node      SKU
master               g1        1         n1-standard-1
n1-standard-2-pool   g1        1         n1-standard-2
```

* **Updating existing NG**

For scenario 1 & 2 we need to update our existing node group. To update existing node group configuration run
the following command.

```yaml
$ pharmer edit nodegroup n1-standard-2-pool -k g

# Please edit the object below. Lines beginning with a '#' will be ignored,
# and an empty file will abort the edit. If an error occurs while saving this file will be
# reopened with the relevant failures.
#
apiVersion: v1alpha1
kind: NodeGroup
metadata:
  clusterName: g1
  creationTimestamp: 2017-11-30T05:40:54Z
  labels:
    node-role.kubernetes.io/node: ""
  name: n1-standard-2-pool
  uid: 089352b2-d591-11e7-b0c0-382c4a73a7c4
spec:
  nodes: 1
  template:
    spec:
      nodeDiskSize: 100
      nodeDiskType: pd-standard
      sku: n1-standard-2
      type: regular
status:
  nodes: 0
```

Here,
* `metadata.name` refers the node group name, which is unique within a cluster.
* `metadata.labels` specifies the label of the nodegroup, which will be add to all nodes of following node group.
    * For master label will be `"node-role.kubernetes.io/master": ""`
    * For node label will be like `"node-role.kubernetes.io/node": ""`
* `metadata.clusterName` indicates the cluster, which has this node group.
* `spec.nodes` shows the number of nodes for this following group.
* `spec.template.spec.nodeDiskSize`: `pharmer` put 100GB by default. You can change it before apply the node group
* `spec.template.spec.nodeDiskType`: `pharmer` put `gp2` as disk type  by default. You can change it before apply the node group
* `spec.template.sku` refers the size of the machine
* `spec.template.spec.type`: `regular` for regular node and `spot` for spot type node
* `status.node` shows the number of nodes that are really present on the current cluster while scaling

To update number of nodes for this nodegroup modify the `node` number under `spec` field.

* **Introduce new NG**
- Regular NG :

To add a new regular node group for an existing cluster you need to run
```console
$ pharmer create ng --nodes=n1-standard-1=1 -k g1

$ pharmer get nodegroup -k g1
NAME                 Cluster   Node      SKU
master               g1        1         n1-standard-1
n1-standard-1-pool   g1        1         n1-standard-1
n1-standard-2-pool   g1        1         n1-standard-

```

You can see the yaml of the newly created node group by running

```yaml
$ pharmer get ng n1-standard-1-pool -k g1 -o yaml
apiVersion: v1alpha1
kind: NodeGroup
metadata:
  clusterName: g1
  creationTimestamp: 2017-11-30T06:12:07Z
  labels:
    node-role.kubernetes.io/node: ""
  name: n1-standard-1-pool
  uid: 64a4fd3c-d595-11e7-80e7-382c4a73a7c4
spec:
  nodes: 1
  template:
    spec:
      nodeDiskSize: 100
      nodeDiskType: pd-standard
      sku: n1-standard-1
      type: regular
status:
  nodes: 0

```
* **Delete existing NG**

If you want delete existing node group following command will help.
```yaml
$ pharmer delete ng n1-standard-2-pool -k g1


$ pharmer get ng n1-standard-2-pool -k g1 -o yaml
apiVersion: v1alpha1
kind: NodeGroup
metadata:
  clusterName: g1
  creationTimestamp: 2017-11-30T05:40:54Z
  deletionTimestamp: 2017-11-30T06:13:29Z
  labels:
    node-role.kubernetes.io/node: ""
  name: n1-standard-2-pool
  uid: 089352b2-d591-11e7-b0c0-382c4a73a7c4
spec:
  nodes: 1
  template:
    spec:
      nodeDiskSize: 100
      nodeDiskType: pd-standard
      sku: n1-standard-2
      type: regular
status:
  nodes: 0
```
Here,

 - `metadata.deletionTimestamp`: will appear if node group deleted command was run

After completing your change on the node groups, you need to apply that via `pharmer` so that changes will be applied
on provider cluster.

```console
$ pharmer apply g1
```

This command will take care of your actions that you applied on the node groups recently.

```console
 $ pharmer get ng -k g1
NAME                 Cluster   Node      SKU
master               g1        1         n1-standard-1
n1-standard-1-pool   g1        1         n1-standard-1
```

### Cluster Upgrading

To upgrade your cluster firstly you need to check if there any update available for your cluster and latest kubernetes version.
To check run:
```console
$ pharmer describe cluster g1

Name:		g1
Version:	v1.11.0
NodeGroup:
  Name                 Node
  ----                 ------
  master               1
  n1-standard-1-pool   1
[upgrade/versions] Cluster version: v1.11.0
[upgrade/versions] kubeadm version: v1.11.0
[upgrade/versions] Latest stable version: v1.11.1
[upgrade/versions] Latest version in the v1.1 series: v1.1.2
Components that will be upgraded after you've upgraded the control plane:
COMPONENT   CURRENT       AVAILABLE
Kubelet     2 x v1.11.0   v1.11.1

Upgrade to the latest stable version:

COMPONENT            CURRENT   AVAILABLE
API Server           v1.11.0   v1.11.1
Controller Manager   v1.11.0   v1.11.1
Scheduler            v1.11.0   v1.11.1
Kube Proxy           v1.11.0   v1.11.1
Kube DNS             1.1.3     1.1.3


You can now apply the upgrade by executing the following command:

	pharmer edit cluster g1 --kubernetes-version=v1.11.1

Note: Before you do can perform this upgrade, you have to update kubeadm to v1.11.1

_____________________________________________________________________

```

Then, if you decided to upgrade you cluster run the command that are showing on describe command.
```console
$ pharmer edit cluster g1 --kubernetes-version=v1.11.1
cluster "g1" updated
```
You can verify your changes by checking the yaml of the cluster.

```yaml
$ pharmer get cluster g1 -o yaml
apiVersion: v1alpha1
kind: Cluster
metadata:
  creationTimestamp: 2017-11-30T05:40:53Z
  generation: 1512023147007799905
  name: g1
  uid: 07fd49c7-d591-11e7-b0c0-382c4a73a7c4
spec:
  api:
    advertiseAddress: ""
    bindPort: 6443
  apiServerExtraArgs:
    cloud-config: /etc/kubernetes/ccm/cloud-config
    kubelet-preferred-address-types: Hostname,InternalDNS,InternalIP,ExternalDNS,ExternalIP
  caCertName: ca
  cloud:
    ccmCredentialName: gce
    cloudProvider: gce
    gce:
      NetworkName: default
      NodeTags:
      - g1-node
    instanceImage: ubuntu-1604-xenial-v20170721
    instanceImageProject: ubuntu-os-cloud
    project: k8s-qa
    region: us-central1
    sshKeyName: g1-fwctge
    zone: us-central1-f
  controllerManagerExtraArgs:
    cloud-config: /etc/kubernetes/ccm/cloud-config
  credentialName: gce
  frontProxyCACertName: front-proxy-ca
  kubernetesVersion: v1.11.1
  networking:
    networkProvider: calico
    nonMasqueradeCIDR: 10.0.0.0/8
    podSubnet: 10.244.0.0/16
status:
  apiServer:
  - address: 10.128.0.2
    type: InternalIP
  - address: 104.154.163.23
    type: ExternalIP
  cloud: {}
  phase: Ready

```

Here, `spec.kubernetesVersion` is changed to `v1.11.0` from `v1.11.1`

If everything looks ok, then run:
```console
$ pharmer apply g1
```

You can check your cluster upgraded or not by running following command on your cluster.
```console
$ kubectl version
Client Version: version.Info{Major:"1", Minor:"11", GitVersion:"v1.11.0", GitCommit:"91e7b4fd31fcd3d5f436da26c980becec37ceefe", GitTreeState:"clean", BuildDate:"2018-06-27T20:17:28Z", GoVersion:"go1.10.2", Compiler:"gc", Platform:"linux/amd64"}
Server Version: version.Info{Major:"1", Minor:"11", GitVersion:"v1.11.0", GitCommit:"91e7b4fd31fcd3d5f436da26c980becec37ceefe", GitTreeState:"clean", BuildDate:"2018-06-27T20:08:34Z", GoVersion:"go1.10.2", Compiler:"gc", Platform:"linux/amd64"}
```


## Cluster Backup

To get a backup of your cluster run the following command:

```console
$ pharmer backup cluster --cluster g1 --backup-dir=g1-backup
```
Here,
   `--backup-dir` is the flag for specifying your backup directory where phamer puts the backup file

After finishing task `pharmer` creates a `.tar.gz` file in your backup directory where you find the backup yaml of your cluster


## Cluster Deleting

To delete your cluster run
```console
$ pharmer delete cluster g1
```
Then, the yaml file looks like

```yaml
 $ pharmer get cluster g1 -o yaml
apiVersion: v1alpha1
kind: Cluster
metadata:
  creationTimestamp: 2017-11-30T05:40:53Z
  deletionTimestamp: 2017-11-30T06:39:34Z
  generation: 1512023147007799905
  name: g1
  uid: 07fd49c7-d591-11e7-b0c0-382c4a73a7c4
spec:
  api:
    advertiseAddress: ""
    bindPort: 6443
  apiServerExtraArgs:
    cloud-config: /etc/kubernetes/ccm/cloud-config
    kubelet-preferred-address-types: Hostname,InternalDNS,InternalIP,ExternalDNS,ExternalIP
  caCertName: ca
  cloud:
    ccmCredentialName: gce
    cloudProvider: gce
    gce:
      NetworkName: default
      NodeTags:
      - g1-node
    instanceImage: ubuntu-1604-xenial-v20170721
    instanceImageProject: ubuntu-os-cloud
    project: k8s-qa
    region: us-central1
    sshKeyName: g1-fwctge
    zone: us-central1-f
  controllerManagerExtraArgs:
    cloud-config: /etc/kubernetes/ccm/cloud-config
  credentialName: gce
  frontProxyCACertName: front-proxy-ca
  kubernetesVersion: v1.11.1
  networking:
    networkProvider: calico
    nonMasqueradeCIDR: 10.0.0.0/8
    podSubnet: 10.244.0.0/16
status:
  apiServer:
  - address: 10.128.0.2
    type: InternalIP
  - address: 104.154.163.23
    type: ExternalIP
  cloud: {}
  phase: Deleting

```

Here,

- `metadata.deletionTimestamp`: is set when cluster deletion command was applied.

Now, to apply delete on provider cluster run
```console
$ pharmer apply g1
```

**Congratulations !!!** , you're an official `pharmer` user now.




