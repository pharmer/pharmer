---
title: Vultr Overview
menu:
  product_pharmer_0.1.0-alpha.2:
    identifier: vultr-overview
    name: Overview
    parent: vultr
    weight: 25
product_name: pharmer
left_menu: product_pharmer_0.1.0-alpha.2
section_menu_id: cloud
url: /products/pharmer/0.1.0-alpha.2/cloud/vultr/
aliases:
  - /products/pharmer/0.1.0-alpha.2/cloud/vultr/README/
---

# Running Kubernetes on Vultr

Following example will use `pharmer ` to create a Kubernetes cluster with 2 worker node servers and a master server (i,e, 3 servers in you cluster).

### Before you start

As a prerequisite, you need to have `pharmer` installed.  To install `pharmer` run the following command.

```console
$ go get github.com/appscode/pharmer
```

### Pharmer storage

To store your cluster  and credential resource, `pharmer` use vfs (virtual file system) as default storage
provider. There is another provider (postgres database) available for storing resources. The configuration
details of the storage provider are specified on `~/.pharmer/config.d/default` file.

 * **Vfs:** By default `pharmer` uses this provider. The configuration details of the file are:
 ```yaml
context: default
kind: PharmerConfig
store:
  local:
    path: /home/sanjid/.pharmer/store.d
```
  Here store type is `local`, so in `path` a local directory is used to locate where the cluster and credential resources will be stored.

  You can also use Amazon's `s3`, `gcs` to use google cloud storage, `azure` or `swift` for storage purpose.
  For using `s3` you have to modify the configuration file with following field
  ```yaml
  s3:
    endpoint: <aws endpoint>
    bucket: <bucket name>
    prefix: <storage prefix>
```
  To use `gcs` modify with
  ```yaml
  gcs:
    bucket: <bucket name>
    prefix: <storage prefix>
```
  For `azure` and `swift` you need to add `container` field along with `prefix` field.


 * **Database:** For storing resources on database `pharmer` uses `postgres` database provider. The configuration
 file will be like :
 ```yaml
context: default
kind: PharmerConfig
store:
  postgres:
    database: <database name>
    user: <database user>
    password: <password>
    host: 127.0.0.1
    port: 5432
```
In this document we will use local file system as a storage provider.

The directory tree of the local storage provider will be look like:

```console
~/.pharmer/
      |--config.d/
      |      |
      |      |__ default (storage configuration file)
      |
      |__ store.d/
             |
             |-- clusters/ (cluster resources)
             |
             |__ credentials/ (credential resources)

```

### Credential importing

To get access on `vultr`, `pharmer` needs credentials of `vultr`. To get the api key go to the **API** section
under **Account** option. Here you see the `Personal Access Token`, copy that key.

![vultr-api-key](/docs/images/vultr/vultr-api-key.jpg)

From command line, run the following command and paste the api key.
```console
$ pharmer create credential vul
```
![vultr-credential](/docs/images/vultr/vultr-credential.png)

Here, `vul` is the credential name, which must be unique within your storage.

To view credential file you can run:
```yaml
$ pharmer get credentials vul -o yaml

  apiVersion: v1alpha1
  kind: Credential
  metadata:
    creationTimestamp: 2017-10-26T11:31:26Z
    name: vul
  spec:
    data:
      token: <your token>
    provider: Vultr
```
Here, `spec.data.token` is the access token that you provided which can be edited by following command:
```console
$ phrmer edit credential vul
```

To see the all credentials you need to run following command.

```console
$ pharmer get credentials
NAME         Provider       Data
vultr        Vultr          token=*****
```

You can also see the stored credential from the following location:
```console
~/.pharmer/store.d/credentials/vul.json
```

You can find other credential operations [here](/docs/credential.md)

### Cluster provisioning

There are two steps to create a Kubernetes cluster using `pharmer`.
In first step `pharmer` create basic configuration file with user choice. Then in second step `pharmer` applies those
information to create cluster on specific provider.

Here, we discuss how to use `pharmer` to create a Kubernetes cluster on `vultr`
 * **Cluster Creating:** We want to create a cluster with following information:
    - Provider: Vultr
    - Cluster name: v1
    - Location: 6 (Atlanta)
    - Number of nodes: 2
    - Node sku: 94 (2048 MB RAM,45 GB SSD,3.00 TB BW)
    - Kubernetes version: 1.8.0
    - Credential name: [vul](#Credential importing)

 For location code and sku details click [hrere](https://github.com/appscode/pharmer/blob/master/data/files/vultr/cloud.json)
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
      --kubeadm-version string      Kubeadm version
      --kubelet-version string      kubelet/kubectl version
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
$ pharmer create cluster v1 \
	--provider=vultr \
	--zone=6 \
	--nodes=94=2 \
	--credential-uid=vul \
	--kubernetes-version=v1.8.0
```
If you want to use a specific version of `kubelet` and `kubeadm` for your cluster, you can pass those flags also.
For example:

`--kubelet-version=1.8.0 --kubeadm-version=1.8.0`

To know about [pod networks](https://kubernetes.io/docs/concepts/cluster-administration/networking/) supports in `pharmer` click [here](/docs/networking.md)

The directory structure of the storage provider will be look like:

```console
~/.pharmer/store.d/clusters/
        |-- v1
        |    |__ nodegroups
        |    |       |__ master.json
        |    |       |
        |    |       |__ 94-pool.json
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
        |          |__ id_v1-jn7bxm
        |          |
        |          |__ id_v1-jn7bxm.pub
        |
        |__ v1.json
```
Here,

   - `/v1/nodegroups/`: contains the node groups information. [Check below](#cluster-scaling) for node group operations.You can see the node group list using following command.
   ```console
$ pharmer get nodegroups -k v1
```
   - `v1/pki`: contains the cluster certificate information containing `ca` and `front-proxy-ca`.
   - `v1/ssh`: has the ssh credentials on cluster's nodes. With this key you can `ssh` into any node on a cluster
   - `v1.json`: contains the cluster resource information

You can view your cluster configuration file by following command.
```yaml
$ pharmer get cluster v1 -o yaml

apiVersion: v1alpha1
kind: Cluster
metadata:
  creationTimestamp: 2017-11-21T07:03:01Z
  generation: 1511247781648715191
  name: v1
  uid: 036ebcb8-ce8a-11e7-bd87-382c4a73a7c4
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
    ccmCredentialName: vultr
    cloudProvider: vultr
    region: "6"
    zone: "6"
  credentialName: vultr
  frontProxyCACertName: front-proxy-ca
  kubernetesVersion: v1.8.0
  networking:
    dnsDomain: cluster.local
    networkProvider: calico
    nonMasqueradeCIDR: 10.0.0.0/8
    podSubnet: 192.168.0.0/16
    serviceSubnet: 10.96.0.0/12
status:
  cloud: {}
  phase: Pending
  sshKeyExternalID: v1-jn7bxm

```
Here,

* `metadata.name` refers the cluster name, which should be unique within your cluster list.
* `metadata.uid` is a unique ACID, which is generated by pharmer
* `spec.cloud` specifies the cloud provider information. pharmer uses `ubuntu-16-04-x64` image by default. don't change the instance images, otherwise cluster may not be working.
* `spec.api.bindPort` is the api server port.
* `spec.networking` specifies the network information of the cluster
    * `networkProvider`: by default it is `calico`. It can be modified by `flannel`.
    * `podSubnet`: in order for network policy to work correctly this field is needed. For flannel it will be `10.244.0.0/16`
* `spec.kubernetesVersion` is the cluster server version. It can be modified.
* `spec.credentialName` is the credential name which is provider during cluster creation command.
* `spec.apiServerExtraArgs` specifies which value will be forwarded to apiserver during cluster installation.
* `spec.authorizationMode` refers the cluster authorization mode
* `status.phase` may be `Pending`, `Ready`, `Deleting`, `Deleted`, `Upgrading` depending on current cluster status.
* `status.sshKeyExternalID` shows which ssh key added to cluster instance.

You can modify this configuration by:
```console
$ pharmer edit cluster v1
```
 * **Applying:** If everything looks ok, we can now apply the resources. This actually creates resources on `Vultr`.
 Up to now we've only been working locally.

 To apply run:
 ```console
$ pharmer apply v1
```
 Now, `pharmer` will apply that configuration, thus create a Kubernetes cluster. After completing task the configuration file of
 the cluster will be look like
```yaml
pharmer get cluster v1 -o yaml
apiVersion: v1alpha1
kind: Cluster
metadata:
  creationTimestamp: 2017-11-21T07:03:01Z
  generation: 1511247781648715191
  name: v1
  uid: 036ebcb8-ce8a-11e7-bd87-382c4a73a7c4
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
    ccmCredentialName: vultr
    cloudProvider: vultr
    instanceImage: "215"
    region: "6"
    zone: "6"
  credentialName: vultr
  frontProxyCACertName: front-proxy-ca
  kubernetesVersion: v1.8.0
  networking:
    dnsDomain: cluster.local
    networkProvider: calico
    nonMasqueradeCIDR: 10.0.0.0/8
    podSubnet: 192.168.0.0/16
    serviceSubnet: 10.96.0.0/12
status:
  apiServer:
  - address: 10.99.0.10
    type: InternalIP
  - address: 45.32.216.208
    type: ExternalIP
  cloud: {}
  phase: Ready
  sshKeyExternalID: v1-jn7bxm
```
Here,

  `status.phase`: is ready. So, you can use your cluster from local machine.

To get the `kubectl` configuration file(kubeconfig) on your local filesystem run the following command.
```console
$ pharmer use cluster v1
```
If you don't have `kubectl` installed click [here](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

Now you can run `kubectl get nodes` and verify that your kubernetes 1.8.0 is running.

```console
$ kubectl get nodes

NAME             STATUS    ROLES     AGE       VERSION
94-pool-hg3pec   Ready     node      22m       v1.8.4
94-pool-nmsg4v   Ready     node      21m       v1.8.4
v1-master        Ready     master    38m       v1.8.4

```

If you want to `ssh` into your instance run the following command
```console
$ pharmer ssh  node v1-master  -k v1
```

### Cluster Scaling

Scaling a cluster refers following meanings:-
 1. Increment the number of nodes of a certain node group
 2. Decrement the number of nodes of a certain node group
 3. Introduce a new node group with a number of nodes
 4. Drop existing node group

To see the current node groups list, you need to run following command:
```console
$ pharmer get nodegroup -k v1
NAME      Cluster   Node      SKU
94-pool   v1        2         94
master    v1        1         95
```

* **Updating existing NG**

For scenario 1 & 2 we need to update our existing node group. To update existing node group configuration run
the following command.

```yaml
$ pharmer edit nodegroup 94-pool  -k v1

# Please edit the object below. Lines beginning with a '#' will be ignored,
# and an empty file will abort the edit. If an error occurs while saving this file will be
# reopened with the relevant failures.
#
apiVersion: v1alpha1
kind: NodeGroup
metadata:
  clusterName: v1
  creationTimestamp: 2017-11-21T07:03:02Z
  labels:
    node-role.kubernetes.io/node: ""
  name: 94-pool
  uid: 03bfee01-ce8a-11e7-bd87-382c4a73a7c4
spec:
  nodes: 2
  template:
    spec:
      sku: "94"
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
$ pharmer create ng --nodes=95=1 -k v1

$ pharmer get nodegroups -k v1
NAME      Cluster   Node      SKU
94-pool   v1        2         94
95-pool   v1        1         95
master    v1        1         95

```
You can see the yaml of newly created node group, you need to run
```yaml
$ pharmer get ng 95-pool -k v1 -o yaml

  apiVersion: v1alpha1
  kind: NodeGroup
  metadata:
    clusterName: v1
    creationTimestamp: 2017-11-21T10:03:24Z
    labels:
      node-role.kubernetes.io/node: ""
    name: 95-pool
    uid: 36843c88-cea3-11e7-981b-382c4a73a7c4
  spec:
    nodes: 1
    template:
      spec:
        sku: "95"
  status:
    nodes: 0
```

* **Delete existing NG**

If you want delete existing node group following command will help.
```yaml
$ pharmer delete ng 94-pool -k v1

$ pharmer get ng 94-pool -k v1 -o yaml
apiVersion: v1alpha1
kind: NodeGroup
metadata:
  clusterName: v1
  creationTimestamp: 2017-11-21T07:03:02Z
  deletionTimestamp: 2017-11-21T10:14:19Z
  labels:
    node-role.kubernetes.io/node: ""
  name: 94-pool
  uid: 03bfee01-ce8a-11e7-bd87-382c4a73a7c4
spec:
  nodes: 2
  template:
    spec:
      sku: "94"
status:
  nodes: 0
```
Here,

 - `metadata.deletionTimestamp`: will appear if node group deleted command was run

After completing your change on the node groups, you need to apply that via `pharmer` so that changes will be applied
on provider cluster.

```console
$ pharmer apply v1
```
This command will take care of your actions that you applied on the node groups recently.

```console
$ pharmer get ng -k v1
NAME      Cluster   Node      SKU
95-pool   v1        1         95
master    v1        1         95
```

### Cluster Upgrading

To upgrade your cluster firstly you need to check if there any update available for your cluster and latest kubernetes version.
To check run:

```console
$ pharmer describe cluster v1
Name:		v1
Version:	1.8.0
NodeGroup:
  Name      Node
  ----      ------
  95-pool   1
  master    1
[upgrade/versions] Cluster version: v1.8.0
[upgrade/versions] kubeadm version: 1.8.4
[upgrade/versions] Latest stable version: v1.8.4
[upgrade/versions] Latest version in the v1.8 series: v1.8.4
Upgrade to the latest version in the v1.8 series:

COMPONENT            CURRENT   AVAILABLE
API Server           v1.8.0    v1.8.4
Controller Manager   v1.8.0    v1.8.4
Scheduler            v1.8.0    v1.8.4
Kube Proxy           v1.8.0    v1.8.4
Kube DNS             1.14.5    1.14.5

You can now apply the upgrade by executing the following command:

	pharmer edit cluster v1 --kubernetes-version=v1.8.4

_____________________________________________________________________

```

Then, if you decided to upgrade you cluster run the command that are showing on describe command.

```console
$ pharmer edit cluster v1 --kubernetes-version=v1.8.4
cluster "v1" updated
```
You can verify your changes by checking the yaml of the cluster.
```yaml
$ pharmer get cluster v1 -o yaml

apiVersion: v1alpha1
kind: Cluster
metadata:
  creationTimestamp: 2017-11-21T07:03:01Z
  generation: 15112634162710````05802
  name: v1
  uid: 036ebcb8-ce8a-11e7-bd87-382c4a73a7c4
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
    ccmCredentialName: vultr
    cloudProvider: vultr
    instanceImage: "215"
    region: "6"
    zone: "6"
  credentialName: vultr
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
  - address: 10.99.0.10
    type: InternalIP
  - address: 45.32.216.208
    type: ExternalIP
  cloud: {}
  phase: Ready
  sshKeyExternalID: v1-jn7bxm

```
Here, `spec.kubernetesVersion` is changed to `v1.8.4` from `v1.8.0`

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
$ pharmer delete cluster v1
```
Then, the yaml file looks like
```yaml
$ pharmer get cluster v1 -o yaml

apiVersion: v1alpha1
kind: Cluster
metadata:
  creationTimestamp: 2017-11-21T07:03:01Z
  deletionTimestamp: 2017-11-21T11:38:28Z
  generation: 1511263416271005802
  name: v1
  uid: 036ebcb8-ce8a-11e7-bd87-382c4a73a7c4
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
    ccmCredentialName: vultr
    cloudProvider: vultr
    instanceImage: "215"
    region: "6"
    zone: "6"
  credentialName: vultr
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
  - address: 10.99.0.10
    type: InternalIP
  - address: 45.32.216.208
    type: ExternalIP
  cloud: {}
  phase: Deleting
  sshKeyExternalID: v1-jn7bxm
```
Here,

- `metadata.deletionTimestamp`: is set when cluster deletion command was applied.

Now, to apply delete on provider cluster run
```console
$ pharmer apply v1
```

**Congratulations !!!** , you're an official `pharmer` user now.
