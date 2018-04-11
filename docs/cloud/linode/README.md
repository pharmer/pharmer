---
title: Linode Overview
menu:
  product_pharmer_0.1.0-alpha.1:
    identifier: linode-overview
    name: Overview
    parent: linode
    weight: 30
product_name: pharmer
menu_name: product_pharmer_0.1.0-alpha.1
section_menu_id: cloud
url: /products/pharmer/0.1.0-alpha.1/cloud/linode/
aliases:
  - /products/pharmer/0.1.0-alpha.1/cloud/linode/README/
---

# Running Kubernetes on [Linode](https://www.linode.com/)

Following example will use `pharmer ` to create a Kubernetes cluster with 2 worker node instances and a master instance (i,e, 3 instance in you cluster).

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
  provider: Linode

```
Here, `spec.data.token` is the access token that you provided which can be edited by following command:
```console
$ phrmer edit credential linode
```

To see the all credentials you need to run following command.

```console
$ pharmer get credentials
NAME         Provider       Data
linode       Linode         token=*****
```

You can also see the stored credential from the following location:
```console
~/.pharmer/store.d/credentials/linode.json
```

You can find other credential operations [here](/docs/credential.md)

### Cluster provisioning

There are two steps to create a Kubernetes cluster using `pharmer`.
In first step `pharmer` create basic configuration file with user choice. Then in second step `pharmer` applies those
information to create cluster on specific provider.

Here, we discuss how to use `pharmer` to create a Kubernetes cluster on `linode`
 * **Cluster Creating:** We want to create a cluster with following information:
    - Provider: Linode
    - Cluster name: l1
    - Location: 3 (Fremont, CA, USA)
    - Number of nodes: 2
    - Node sku: 1 (Linode 1024)
    - Kubernetes version: 1.9.0
    - Credential name: [linode](#credential-importing)

For location code and sku details click [hrere](https://github.com/pharmer/pharmer/blob/master/data/files/linode/cloud.json)
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
       --networking string           Networking mode to use. calico(default), flannel (default "calico")
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
$ pharmer create cluster l1 \
	--provider=linode \
	--zone=3 \
	--nodes=1=2 \
	--credential-uid=linode \
	--kubernetes-version=v1.9.0
```

To know about [pod networks](https://kubernetes.io/docs/concepts/cluster-administration/networking/) supports in `pharmer` click [here](/docs/networking.md)

The directory structure of the storage provider will be look like:

```console
~/.pharmer/store.d/clusters/
        |-- v1
        |    |__ nodegroups
        |    |       |__ master.json
        |    |       |
        |    |       |__ 1-pool.json
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
        |          |__ id_l1-25woji
        |          |
        |          |__ id_l1-25woji.pub
        |
        |__ l1.json
```
Here,

   - `/v1/nodegroups/`: contains the node groups information. [This](#Cluster scalling) describes node groups operation.You can see the node group list using following command.
   ```console
$ pharmer get nodegroups -k v1
```
   - `v1/pki`: contains the cluster certificate information containing `ca` and `front-proxy-ca`.
   - `v1/ssh`: has the ssh credentials on cluster's nodes. With this key you can `ssh` into any node on a cluster
   - `v1.json`: contains the cluster resource information
You can view your cluster configuration file by following command.
```yaml
$ pharmer get cluster l1 -o yaml
apiVersion: v1alpha1
kind: Cluster
metadata:
  creationTimestamp: 2017-11-22T09:15:40Z
  generation: 1511342140472556499
  name: l1
  uid: b5aca10d-cf65-11e7-a3f3-382c4a73a7c4
spec:
  api:
    advertiseAddress: ""
    bindPort: 6443
  apiServerExtraArgs:
    kubelet-preferred-address-types: InternalIP,ExternalIP
  authorizationModes:
  - Node
  - RBAC
  caCertName: ca
  cloud:
    ccmCredentialName: linode
    cloudProvider: linode
    linode:
      rootPassword: YueW_Qam8eUHQvws
    region: "3"
    zone: "3"
  credentialName: linode
  frontProxyCACertName: front-proxy-ca
  kubernetesVersion: v1.9.0
  networking:
    dnsDomain: cluster.local
    networkProvider: calico
    nonMasqueradeCIDR: 10.0.0.0/8
    podSubnet: 192.168.0.0/16
    serviceSubnet: 10.96.0.0/12
status:
  cloud: {}
  phase: Pending
  sshKeyExternalID: l1-25woji
```
Here,

* `metadata.name` refers the cluster name, which should be unique within your cluster list.
* `metadata.uid` is a unique ACID, which is generated by pharmer
* `spec.cloud` specifies the cloud provider information. pharmer uses `ubuntu-16-04-x64` image by default. don't change the instance images, otherwise cluster may not be working.
* `spec.api.bindPort` is the api server port.
* `spec.networking` specifies the network information of the cluster
    * `networkProvider`: by default it is `calico`. To modify it click [here](/docs/networking.md).
    * `podSubnet`: in order for network policy to work correctly this field is needed. For flannel it will be `10.244.0.0/16`
* `spec.kubernetesVersion` is the cluster server version. It can be modified.
* `spec.credentialName` is the credential name which is provider during cluster creation command.
* `spec.apiServerExtraArgs` specifies which value will be forwarded to apiserver during cluster installation.
* `spec.authorizationMode` refers the cluster authorization mode
* `status.phase` may be `Pending`, `Ready`, `Deleting`, `Deleted`, `Upgrading` depending on current cluster status.
* `status.sshKeyExternalID` shows which ssh key added to cluster instance.

You can modify this configuration by:
```console
$ pharmer edit cluster l1
```
* **Applying:** If everything looks ok, we can now apply the resources. This actually creates resources on `Linode`.
 Up to now we've only been working locally.

 To apply run:
 ```console
$ pharmer apply l1
```
 Now, `pharmer` will apply that configuration, thus create a Kubernetes cluster. After completing task the configuration file of
 the cluster will be look like
```yaml
pharmer get cluster l1 -o yaml
apiVersion: v1alpha1
kind: Cluster
metadata:
  creationTimestamp: 2017-11-22T09:15:40Z
  generation: 1511342140472556499
  name: l1
  uid: b5aca10d-cf65-11e7-a3f3-382c4a73a7c4
spec:
  api:
    advertiseAddress: ""
    bindPort: 6443
  apiServerExtraArgs:
    kubelet-preferred-address-types: InternalIP,ExternalIP
  authorizationModes:
  - Node
  - RBAC
  caCertName: ca
  cloud:
    ccmCredentialName: linode
    cloudProvider: linode
    instanceImage: "146"
    linode:
      kernelId: 138
      rootPassword: YueW_Qam8eUHQvws
    region: "3"
    zone: "3"
  credentialName: linode
  frontProxyCACertName: front-proxy-ca
  kubernetesVersion: v1.9.0
  networking:
    dnsDomain: cluster.local
    networkProvider: calico
    nonMasqueradeCIDR: 10.0.0.0/8
    podSubnet: 192.168.0.0/16
    serviceSubnet: 10.96.0.0/12
status:
  apiServer:
  - address: 192.168.196.71
    type: InternalIP
  - address: 198.74.51.221
    type: ExternalIP
  cloud: {}
  phase: Ready
  sshKeyExternalID: l1-25woji
```

Here,

  `status.phase`: is ready. So, you can use your cluster from local machine.

To get the `kubectl` configuration file(kubeconfig) on your local filesystem run the following command.
```console
$ pharmer use cluster l1
```
If you don't have `kubectl` installed click [here](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

Now you can run `kubectl get nodes` and verify that your kubernetes 1.9.0 is running.

```console
$ kubectl get nodes

NAME                 STATUS    ROLES     AGE       VERSION
l1-023-239-003-219   Ready     node      48s       v1.8.4
l1-104-200-024-144   Ready     node      35s       v1.8.4
l1-198-074-051-221   Ready     master    6m        v1.8.4
```

If you want to `ssh` into your instance run the following command
```console
$ pharmer ssh  node l1-198-074-051-221  -k l1
```

### Cluster Scaling

Scaling a cluster refers following meanings:-
 1. Increment the number of nodes of a certain node group
 2. Decrement the number of nodes of a certain node group
 3. Introduce a new node group with a number of nodes
 4. Drop existing node group

To see the current node groups list, you need to run following command:
```console
$ pharmer get nodegroup -k l1
NAME      Cluster   Node      SKU
1-pool    l1        2         1
master    l1        1         3
```
* **Updating existing NG**

For scenario 1 & 2 we need to update our existing node group. To update existing node group configuration run
the following command.

```yaml
$ pharmer edit nodegroup 1-pool -k l1

# Please edit the object below. Lines beginning with a '#' will be ignored,
# and an empty file will abort the edit. If an error occurs while saving this file will be
# reopened with the relevant failures.
#
apiVersion: v1alpha1
kind: NodeGroup
metadata:
  clusterName: l1
  creationTimestamp: 2017-11-22T09:15:40Z
  labels:
    node-role.kubernetes.io/node: ""
  name: 1-pool
  uid: b5e7c87f-cf65-11e7-a3f3-382c4a73a7c4
spec:
  nodes: 2
  template:
    spec:
      sku: "1"
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
$ pharmer create ng --nodes=2=1 -k l1

$ pharmer get nodegroups -k l1
NAME      Cluster   Node      SKU
1-pool    l1        2         1
2-pool    l1        1         2
master    l1        1         3

```
You can see the yaml of newly created node group, you need to run
```yaml
$ pharmer get ng 2-pool -k l1 -o yaml
  apiVersion: v1alpha1
  kind: NodeGroup
  metadata:
    clusterName: l1
    creationTimestamp: 2017-11-22T10:26:27Z
    labels:
      node-role.kubernetes.io/node: ""
    name: 2-pool
    uid: 991f8e0e-cf6f-11e7-bb02-382c4a73a7c4
  spec:
    nodes: 1
    template:
      spec:
        sku: "2"
  status:
    nodes: 0
```


* **Delete existing NG**

If you want delete existing node group following command will help.
```yaml
$ pharmer delete ng 1-pool -k l1

$ pharmer get ng 1-pool -k l1 -o yaml
  apiVersion: v1alpha1
  kind: NodeGroup
  metadata:
    clusterName: l1
    creationTimestamp: 2017-11-22T09:15:40Z
    deletionTimestamp: 2017-11-22T10:29:23Z
    labels:
      node-role.kubernetes.io/node: ""
    name: 1-pool
    uid: b5e7c87f-cf65-11e7-a3f3-382c4a73a7c4
  spec:
    nodes: 2
    template:
      spec:
        sku: "1"
  status:
    nodes: 0
```
Here,

 - `metadata.deletionTimestamp`: will appear if node group deleted command was run

After completing your change on the node groups, you need to apply that via `pharmer` so that changes will be applied
on provider cluster.

```console
$ pharmer apply l1
```
This command will take care of your actions that you applied on the node groups recently.

```console
$ pharmer get ng -k l1
NAME      Cluster   Node      SKU
2-pool    l1        1         2
master    l1        1         3
```

### Cluster Upgrading

To upgrade your cluster firstly you need to check if there any update available for your cluster and latest kubernetes version.
To check run:
```console
$ pharmer describe cluster l1
Name:		l1
Version:	v1.9.0
NodeGroup:
  Name     Node
  ----     ------
  2-pool   1
  master   1
[upgrade/versions] Cluster version: v1.9.0
[upgrade/versions] kubeadm version: v1.8.4
[upgrade/versions] Latest stable version: v1.8.4
[upgrade/versions] Latest version in the v1.8 series: v1.8.4
Upgrade to the latest version in the v1.8 series:

COMPONENT            CURRENT   AVAILABLE
API Server           v1.9.0    v1.8.4
Controller Manager   v1.9.0    v1.8.4
Scheduler            v1.9.0    v1.8.4
Kube Proxy           v1.9.0    v1.8.4
Kube DNS             1.14.5    1.14.5

You can now apply the upgrade by executing the following command:

	pharmer edit cluster l1 --kubernetes-version=v1.8.4

_____________________________________________________________________
```

Then, if you decided to upgrade you cluster run the command that are showing on describe command.
```console
$ pharmer edit cluster l1 --kubernetes-version=v1.8.4
cluster "l1" updated
```
You can verify your changes by checking the yaml of the cluster.
```console
$ pharmer get cluster l1 -o yaml
apiVersion: v1alpha1
kind: Cluster
metadata:
  creationTimestamp: 2017-11-22T09:15:40Z
  generation: 1511347216320193468
  name: l1
  uid: b5aca10d-cf65-11e7-a3f3-382c4a73a7c4
spec:
  api:
    advertiseAddress: ""
    bindPort: 6443
  apiServerExtraArgs:
    kubelet-preferred-address-types: InternalIP,ExternalIP
  authorizationModes:
  - Node
  - RBAC
  caCertName: ca
  cloud:
    ccmCredentialName: linode
    cloudProvider: linode
    instanceImage: "146"
    linode:
      kernelId: 138
      rootPassword: YueW_Qam8eUHQvws
    region: "3"
    zone: "3"
  credentialName: linode
  frontProxyCACertName: front-proxy-ca
  kubernetesVersion: v1.8.4
  networking:
    dnsDomain: cluster.local
    networkProvider: calico
    nonMasqueradeCIDR: 10.0.0.0/8
    podSubnet: 192.168.0.0/16
    serviceSubnet: 10.96.0.0/12
status:
  apiServer:
  - address: 192.168.196.71
    type: InternalIP
  - address: 198.74.51.221
    type: ExternalIP
  cloud: {}
  phase: Ready
  sshKeyExternalID: l1-25woji
```
Here, `spec.kubernetesVersion` is changed to `v1.8.4` from `v1.9.0`

If everything looks ok, then run:
```console
$ pharmer apply v1
```
You can check your cluster upgraded or not by running following command on your cluster.
```console
$ kubectl version
Client Version: version.Info{Major:"1", Minor:"8", GitVersion:"v1.8.4", GitCommit:"9befc2b8928a9426501d3bf62f72849d5cbcd5a3", GitTreeState:"clean", BuildDate:"2017-11-20T05:28:34Z", GoVersion:"go1.8.3", Compiler:"gc", Platform:"linux/amd64"}
Server Version: version.Info{Major:"1", Minor:"8", GitVersion:"v1.8.4", GitCommit:"9befc2b8928a9426501d3bf62f72849d5cbcd5a3", GitTreeState:"clean", BuildDate:"2017-11-20T05:17:43Z", GoVersion:"go1.8.3", Compiler:"gc", Platform:"linux/amd64"}
```
## Cluster Deleting

To delete your cluster run
```console
$ pharmer delete cluster l1
```
Then, the yaml file looks like

```yaml
$ pharmer get cluster l1 -o yaml

apiVersion: v1alpha1
kind: Cluster
metadata:
  creationTimestamp: 2017-11-22T09:15:40Z
  deletionTimestamp: 2017-11-22T10:43:24Z
  generation: 1511347216320193468
  name: l1
  uid: b5aca10d-cf65-11e7-a3f3-382c4a73a7c4
spec:
  api:
    advertiseAddress: ""
    bindPort: 6443
  apiServerExtraArgs:
    kubelet-preferred-address-types: InternalIP,ExternalIP
  authorizationModes:
  - Node
  - RBAC
  caCertName: ca
  cloud:
    ccmCredentialName: linode
    cloudProvider: linode
    instanceImage: "146"
    linode:
      kernelId: 138
      rootPassword: YueW_Qam8eUHQvws
    region: "3"
    zone: "3"
  credentialName: linode
  frontProxyCACertName: front-proxy-ca
  kubernetesVersion: v1.8.4
  networking:
    dnsDomain: cluster.local
    networkProvider: calico
    nonMasqueradeCIDR: 10.0.0.0/8
    podSubnet: 192.168.0.0/16
    serviceSubnet: 10.96.0.0/12
status:
  apiServer:
  - address: 192.168.196.71
    type: InternalIP
  - address: 198.74.51.221
    type: ExternalIP
  cloud: {}
  phase: Deleting
  sshKeyExternalID: l1-25woji
```
Here,

- `metadata.deletionTimestamp`: is set when cluster deletion command was applied.

Now, to apply delete on provider cluster run
```console
$ pharmer apply l1
```

**Congratulations !!!** , you're an official `pharmer` user now.

