---
title: Lightsail Overview
menu:
  product_pharmer_0.1.0-alpha.1:
    identifier: lightsail-overview
    name: Overview
    parent: lightsail
    weight: 10
product_name: pharmer
menu_name: product_pharmer_0.1.0-alpha.1
section_menu_id: cloud
url: /products/pharmer/0.1.0-alpha.1/cloud/lightsail/
aliases:
  - /products/pharmer/0.1.0-alpha.1/cloud/lightsail/README/
---

# Running Kubernetes on [Amazon Lightsail](https://amazonlightsail.com/)

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

You can use your [aws credential](/docs/cloud/aws/README.md#credential-importing) to creat cluster in lightsail.

### Cluster provisioning

There are two steps to create a Kubernetes cluster using `pharmer`.
In first step `pharmer` create basic configuration file with user choice. Then in second step `pharmer` applies those
information to create cluster on specific provider.

Here, we discuss how to use `pharmer` to create a Kubernetes cluster on `lightsail`
 * **Cluster Creating:** We want to create a cluster with following information:
    - Provider: Lightsail
    - Cluster name: lsx
    - Location: us-west-2a (Oregon)
    - Number of nodes: 1
    - Node sku: small_1_0
    - Kubernetes version: 1.11.0
    - Credential name: [aws](/docs/cloud/aws/README.md#credential-importing)

For location code and sku details click [hrere](https://github.com/pharmer/pharmer/blob/master/data/files/lightsail/cloud.json)
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

$ pharmer create cluster lsx \
	--provider=lightsail \
	--zone=us-west-2a \
	--nodes=small_1_0=1 \
	--credential-uid=aws \
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
        |    |       |__ small-1-0-pool.json
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
        |          |__ id_lsx-fzx7yg
        |          |
        |          |__ id_lsx-fzx7yg.pub
        |
        |__ lsx.json
```
Here,

   - `/v1/nodegroups/`: contains the node groups information. [Check below](#cluster-scaling) for node group operations.You can see the node group list using following command.
   ```console
$ pharmer get nodegroups -k lsx
```
   - `v1/pki`: contains the cluster certificate information containing `ca` and `front-proxy-ca`.
   - `v1/ssh`: has the ssh credentials on cluster's nodes. With this key you can `ssh` into any node on a cluster
   - `v1.json`: contains the cluster resource information
You can view your cluster configuration file by following command.
```yaml
$ pharmer get cluster lsx -o yaml
apiVersion: v1alpha1
kind: Cluster
metadata:
  creationTimestamp: 2017-11-28T10:51:53Z
  generation: 1511866313656558299
  name: lsx
  uid: 253d191b-d42a-11e7-b9e5-382c4a73a7c4
spec:
  api:
    advertiseAddress: ""
    bindPort: 6443
  apiServerExtraArgs:
    enable-admission-plugins: Initializers,NodeRestriction,NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,ValidatingAdmissionWebhook,DefaultTolerationSeconds,MutatingAdmissionWebhook,ResourceQuota
    kubelet-preferred-address-types: InternalIP,ExternalIP
    runtime-config: admissionregistration.k8s.io/v1alpha1
  caCertName: ca
  cloud:
    ccmCredentialName: aws
    cloudProvider: lightsail
    region: us-west-2
    sshKeyName: lsx-fzx7yg
    zone: us-west-2a
  credentialName: aws
  frontProxyCACertName: front-proxy-ca
  kubernetesVersion: v1.11.0
  networking:
    dnsDomain: cluster.local
    networkProvider: calico
    podSubnet: 192.168.0.0/16
    serviceSubnet: 10.96.0.0/12
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
$ pharmer edit cluster lsx
```

* **Applying:** If everything looks ok, we can now apply the resources. This actually creates resources on `Lightsail`.
 Up to now we've only been working locally.

 To apply run:
 ```console
$ pharmer apply lsx
```

Now, `pharmer` will apply that configuration, thus create a Kubernetes cluster. After completing task the configuration file of
the cluster will be look like

```yaml
apiVersion: v1alpha1
kind: Cluster
metadata:
  creationTimestamp: 2017-11-28T10:51:53Z
  generation: 1511866313656558299
  name: lsx
  uid: 253d191b-d42a-11e7-b9e5-382c4a73a7c4
spec:
  api:
    advertiseAddress: ""
    bindPort: 6443
  apiServerExtraArgs:
    enable-admission-plugins: Initializers,NodeRestriction,NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,ValidatingAdmissionWebhook,DefaultTolerationSeconds,MutatingAdmissionWebhook,ResourceQuota
    kubelet-preferred-address-types: InternalIP,ExternalIP
    runtime-config: admissionregistration.k8s.io/v1alpha1
  caCertName: ca
  cloud:
    ccmCredentialName: aws
    cloudProvider: lightsail
    instanceImage: ubuntu_16_04_1
    region: us-west-2
    sshKeyName: lsx-fzx7yg
    zone: us-west-2a
  credentialName: aws
  frontProxyCACertName: front-proxy-ca
  kubernetesVersion: v1.11.0
  networking:
    dnsDomain: cluster.local
    networkProvider: calico
    podSubnet: 192.168.0.0/16
    serviceSubnet: 10.96.0.0/12
status:
  apiServer:
  - address: 172.26.10.22
    type: InternalIP
  - address: 52.39.175.84
    type: ExternalIP
  cloud: {}
  phase: Ready
```

Here,

  `status.phase`: is ready. So, you can use your cluster from local machine.

To get the `kubectl` configuration file(kubeconfig) on your local filesystem run the following command.
```console
$ pharmer use cluster lsx
```
If you don't have `kubectl` installed click [here](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

Now you can run `kubectl get nodes` and verify that your kubernetes 1.11.0 is running.

```console
$ kubectl get nodes

NAME              STATUS    AGE       VERSION
ip-172-26-10-22   Ready     5m        v1.11.0
ip-172-26-2-245   Ready     2m        v1.11.0
```

If you want to `ssh` into your instance run the following command
```console
$ pharmer ssh node ip-172-26-10-22 -k lsx
```

### Cluster Scaling

Scaling a cluster refers following meanings:-
 1. Increment the number of nodes of a certain node group
 2. Decrement the number of nodes of a certain node group
 3. Introduce a new node group with a number of nodes
 4. Drop existing node group

To see the current node groups list, you need to run following command:
```console
$ pharmer get nodegroup -k lsx
NAME             Cluster   Node      SKU
master           lsx       1         small_1_0
small-1-0-pool   lsx       1         small_1_0
```

* **Updating existing NG**

For scenario 1 & 2 we need to update our existing node group. To update existing node group configuration run
the following command.

```yaml
$ pharmer edit nodegroup small-1-0-pool -k lsx

# Please edit the object below. Lines beginning with a '#' will be ignored,
# and an empty file will abort the edit. If an error occurs while saving this file will be
# reopened with the relevant failures.
#
apiVersion: v1alpha1
kind: NodeGroup
metadata:
  clusterName: lsx
  creationTimestamp: 2017-11-28T10:51:54Z
  labels:
    node-role.kubernetes.io/node: ""
  name: small-1-0-pool
  uid: 25fa172d-d42a-11e7-b9e5-382c4a73a7c4
spec:
  nodes: 1
  template:
    spec:
      sku: small_1_0
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
* `spec.template.sku` refers the size of the machine
* `status.node` shows the number of nodes that are really present on the current cluster while scaling

To update number of nodes for this nodegroup modify the `node` number under `spec` field.

* **Introduce new NG**

To add a new node group for an existing cluster you need to run

```console
$ pharmer create ng --nodes=small_1_1=0 -k lsx

$ pharmer get nodegroup -k lsx
  NAME             Cluster   Node      SKU
  master           lsx       1         small_1_0
  small-1-0-pool   lsx       2         small_1_0
  small-1-1-pool   lsx       0         small_1_
```

You can see the yaml of newly created node group, you need to run
```yaml
$ pharmer get nodegroup small-1-1-pool -o yaml -k lsx
apiVersion: v1alpha1
kind: NodeGroup
metadata:
  clusterName: lsx
  creationTimestamp: 2017-11-28T11:21:32Z
  labels:
    node-role.kubernetes.io/node: ""
  name: small-1-1-pool
  uid: 495d502d-d42e-11e7-a1e2-382c4a73a7c4
spec:
  nodes: 0
  template:
    spec:
      sku: small_1_1
      type: regular
status:
  nodes: 0
```
* **Delete existing NG**

If you want delete existing node group following command will help.

```yaml
$ pharmer delete ng small-1-1-pool -k lsx

$ pharmer get nodegroup small-1-1-pool -o yaml -k lsx
apiVersion: v1alpha1
kind: NodeGroup
metadata:
  clusterName: lsx
  creationTimestamp: 2017-11-28T11:21:32Z
  deletionTimestamp: 2017-11-28T11:23:05Z
  labels:
    node-role.kubernetes.io/node: ""
  name: small-1-1-pool
  uid: 495d502d-d42e-11e7-a1e2-382c4a73a7c4
spec:
  nodes: 0
  template:
    spec:
      sku: small_1_1
      type: regular
status:
  nodes: 0
```

Here,

 - `metadata.deletionTimestamp`: will appear if node group deleted command was run

After completing your change on the node groups, you need to apply that via `pharmer` so that changes will be applied
on provider cluster.

```console
$ pharmer apply lsx
```

This command will take care of your actions that you applied on the node groups recently.

```console
 $ pharmer get ng -k lsx
NAME             Cluster   Node      SKU
master           lsx       1         small_1_0
small-1-0-pool   lsx       2         small_1_0
```

### Cluster Upgrading

To upgrade your cluster firstly you need to check if there any update available for your cluster and latest kubernetes version.
To check run

```console

$ pharmer describe cluster lsx
Name:		lsx
Version:	v1.11.0
NodeGroup:
  Name             Node
  ----             ------
  master           1
  small-1-0-pool   2
[upgrade/versions] Cluster version: v1.11.0
[upgrade/versions] kubeadm version: v1.11.0
[upgrade/versions] Latest stable version: v1.11.1
[upgrade/versions] Latest version in the v1.1 series: v1.1.8
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

	pharmer edit cluster lsx --kubernetes-version=v1.11.1

_____________________________________________________________________

```
Then, if you decided to upgrade you cluster run the command that are showing on describe command.
```console
$ pharmer edit cluster lsx --kubernetes-version=v1.11.1
cluster "lsx" updated



You can verify your changes by checking the yaml of the cluster.
```yaml
$ pharmer get cluster lsx -o yaml
apiVersion: v1alpha1
kind: Cluster
metadata:
  creationTimestamp: 2017-11-28T10:51:53Z
  generation: 1511868490935037825
  name: lsx
  uid: 253d191b-d42a-11e7-b9e5-382c4a73a7c4
spec:
  api:
    advertiseAddress: ""
    bindPort: 6443
  apiServerExtraArgs:
    enable-admission-plugins: Initializers,NodeRestriction,NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,ValidatingAdmissionWebhook,DefaultTolerationSeconds,MutatingAdmissionWebhook,ResourceQuota
    kubelet-preferred-address-types: InternalIP,ExternalIP
    runtime-config: admissionregistration.k8s.io/v1alpha1
  caCertName: ca
  cloud:
    ccmCredentialName: aws
    cloudProvider: lightsail
    instanceImage: ubuntu_16_04_1
    region: us-west-2
    sshKeyName: lsx-fzx7yg
    zone: us-west-2a
  credentialName: aws
  frontProxyCACertName: front-proxy-ca
  kubernetesVersion: v1.11.1
  networking:
    dnsDomain: cluster.local
    networkProvider: calico
    podSubnet: 192.168.0.0/16
    serviceSubnet: 10.96.0.0/12
status:
  apiServer:
  - address: 172.26.10.22
    type: InternalIP
  - address: 52.39.175.84
    type: ExternalIP
  cloud: {}
  phase: Ready

```
Here, `spec.kubernetesVersion` is changed to `v1.11.0` from `v1.11.1`

If everything looks ok, then run:
```console
$ pharmer apply lsx
```

You can check your cluster upgraded or not by running following command on your cluster.
```console
$ kubectl version
Client Version: version.Info{Major:"1", Minor:"11", GitVersion:"v1.11.0", GitCommit:"91e7b4fd31fcd3d5f436da26c980becec37ceefe", GitTreeState:"clean", BuildDate:"2018-06-27T20:17:28Z", GoVersion:"go1.10.2", Compiler:"gc", Platform:"linux/amd64"}
Server Version: version.Info{Major:"1", Minor:"11", GitVersion:"v1.11.0", GitCommit:"91e7b4fd31fcd3d5f436da26c980becec37ceefe", GitTreeState:"clean", BuildDate:"2018-06-27T20:08:34Z", GoVersion:"go1.10.2", Compiler:"gc", Platform:"linux/amd64"}
```
## Cluster Deleting

To delete your cluster run
```console
$ pharmer delete cluster lsx
```
Then, the yaml file looks like

```yaml
$ pharmer get cluster lsx -o yaml
apiVersion: v1alpha1
kind: Cluster
metadata:
  creationTimestamp: 2017-11-28T10:51:53Z
  deletionTimestamp: 2017-11-28T11:29:36Z
  generation: 1511868490935037825
  name: lsx
  uid: 253d191b-d42a-11e7-b9e5-382c4a73a7c4
spec:
  api:
    advertiseAddress: ""
    bindPort: 6443
  apiServerExtraArgs:
    enable-admission-plugins: Initializers,NodeRestriction,NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,ValidatingAdmissionWebhook,DefaultTolerationSeconds,MutatingAdmissionWebhook,ResourceQuota
    kubelet-preferred-address-types: InternalIP,ExternalIP
    runtime-config: admissionregistration.k8s.io/v1alpha1
  caCertName: ca
  cloud:
    ccmCredentialName: aws
    cloudProvider: lightsail
    instanceImage: ubuntu_16_04_1
    region: us-west-2
    sshKeyName: lsx-fzx7yg
    zone: us-west-2a
  credentialName: aws
  frontProxyCACertName: front-proxy-ca
  kubernetesVersion: v1.11.1
  networking:
    dnsDomain: cluster.local
    networkProvider: calico
    podSubnet: 192.168.0.0/16
    serviceSubnet: 10.96.0.0/12
status:
  apiServer:
  - address: 172.26.10.22
    type: InternalIP
  - address: 52.39.175.84
    type: ExternalIP
  cloud: {}
  phase: Deleting

```
Here,

- `metadata.deletionTimestamp`: is set when cluster deletion command was applied.

Now, to apply delete on provider cluster run
```console
$ pharmer apply lsx
```

**Congratulations !!!** , you're an official `pharmer` user now.








