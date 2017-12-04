---
title: DigitalOcean Overview
menu:
  product_pharmer_0.1.0-alpha.1:
    identifier: digitalocean-overview
    name: Overview
    parent: digital-ocean
    weight: 10
product_name: pharmer
menu_name: product_pharmer_0.1.0-alpha.1
section_menu_id: cloud
url: /products/pharmer/0.1.0-alpha.1/cloud/digitalocean/
aliases:
  - /products/pharmer/0.1.0-alpha.1/cloud/digitalocean/README/
---
# Running Kubernetes on [DigitalOcean](https://cloud.digitalocean.com)

Following example will use `pharmer ` to create a Kubernetes cluster with 2 worker node droplets and a master droplet (i,e, 3 droplets in you cluster).

### Before you start

As a prerequisite, you need to have `pharmer` installed.  To install `pharmer` run the following command.
```console
$ go get github.com/pharmer/pharmer
```

### Pharmer storage

To store your cluster  and credential resource, `pharmer` use [vfs](/docs/cli/vfs.md) as default storage
provider. There is another provider [postgres database](/docs/cli/xorm.md) available for storing resources.

To know more click [here](/docs/cli/datastore.md)

In this document we will use local file system ([vfs](/docs/cli/vfs.md)) as a storage provider.


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
  provider: DigitalOcean
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
do           DigitalOcean   token=*****
```
You can also see the stored credential from the following location:
```console
~/.pharmer/store.d/credentials/do.json            
```

You can find other credential operations [here](/docs/credential.md)

### Cluster provisioning

There are two steps to create a Kubernetes cluster using `pharmer`.
In first step `pharmer` create basic configuration file with user choice. Then in second step `pharmer` applies those 
information to create cluster on specific provider. 

Here, we discuss how to use `pharmer` to create a Kubernetes cluster on `digitalocean`
 * **Cluster Creating:** We want to create a cluster with following information:
    - Provider: DigitalOcean
    - Cluster name: d1
    - Location: nyc3 (New York)
    - Number of nodes: 2
    - Node sku: 2gb
    - Kubernetes version: 1.8.0
    - Credential name: [do](#credential-importing)

For location code and sku details click [hrere](https://github.com/pharmer/pharmer/blob/master/data/files/digitalocean/cloud.json)   
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
$ pharmer create cluster d1 \
	--provider=digitalocean \
	--zone=nyc3 \
	--nodes=2gb=2 \
	--credential-uid=do \
	--kubernetes-version=v1.8.0
```
If you want to use a specific version of `kubelet` and `kubeadm` for your cluster, you can pass those flags also.
For example:

`--kubelet-version=1.8.0 --kubeadm-version=1.8.0`

To know about [pod networks](https://kubernetes.io/docs/concepts/cluster-administration/networking/) supports in `pharmer` click [here](/docs/networking.md)

The directory structure of the storage provider will be look like:

```console
~/.pharmer/store.d/clusters/
        |-- d1
        |    |__ nodegroups
        |    |       |__ master.json
        |    |       |
        |    |       |__ 2gb-pool.json
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
        |          |__ id_d1-fpw40n
        |          |
        |          |__ id_f1-fpw40n.pub
        |
        |__ d1.json
```

Here,

   - `/v1/nodegroups/`: contains the node groups information. [Check below](#cluster-scaling) for node group operations.You can see the node group list using following command.
   ```console
$ pharmer get nodegroups -k d1
```
   - `v1/pki`: contains the cluster certificate information containing `ca` and `front-proxy-ca`.
   - `v1/ssh`: has the ssh credentials on cluster's nodes. With this key you can `ssh` into any node on a cluster
   - `v1.json`: contains the cluster resource information
You can view your cluster configuration file by following command.

```yaml
$ pharmer get cluster d1 -o yaml

apiVersion: v1alpha1
kind: Cluster
metadata:
  creationTimestamp: 2017-11-10T11:26:23Z
  generation: 1510313183635265805
  name: d1
  uid: ACID-K8S-C-v4ec7ho3if3tt5p
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
    cloudProvider: digitalocean
    instanceImage: ubuntu-16-04-x64
    region: nyc3
    sshKeyName: d1-fpw40n
    zone: nyc3
  credentialName: do
  frontProxyCACertName: front-proxy-ca
  kubernetesVersion: v1.8.0
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
$ pharmer edit cluster d1
```
* **Applying:** If everything looks ok, we can now apply the resources. This actually creates resources on `DigitalOcean`.
 Up to now we've only been working locally.

 To apply run:
 ```console
$ pharmer apply d1
```
 Now, `pharmer` will apply that configuration, thus create a Kubernetes cluster. After completing task the configuration file of
 the cluster will be look like
```yaml
$ pharmer get cluster d1 -o yaml
apiVersion: v1alpha1
kind: Cluster
metadata:
  creationTimestamp: 2017-11-10T11:26:23Z
  generation: 1510313183635265805
  name: d1
  uid: ACID-K8S-C-v4ec7ho3if3tt5p
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
    cloudProvider: digitalocean
    instanceImage: ubuntu-16-04-x64
    region: nyc3
    sshKeyName: d1-fpw40n
    zone: nyc3
  credentialName: do
  frontProxyCACertName: front-proxy-ca
  kubernetesVersion: v1.8.0
  networking:
    dnsDomain: cluster.local
    networkProvider: calico
    podSubnet: 192.168.0.0/16
    serviceSubnet: 10.96.0.0/12
status:
  apiServer:
  - address: 10.132.42.222
    type: InternalIP
  - address: 159.203.97.84
    type: ExternalIP
  cloud:
    sshKeyExternalID: "15733879"
  phase: Ready

```
Here,

  `status.phase`: is ready. So, you can use your cluster from local machine.

To get the `kubectl` configuration file(kubeconfig) on your local filesystem run the following command.
```console
$ pharmer use cluster d1
```
If you don't have `kubectl` installed click [here](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

Now you can run `kubectl get nodes` and verify that your kubernetes 1.8.0 is running.

```console
$ kubectl get nodes

NAME              STATUS    ROLES     AGE       VERSION
2gb-pool-oqybzs   Ready     node      40s       v1.8.4
2gb-pool-qqluix   Ready     node      34s       v1.8.4
d1-master         Ready     master    3m        v1.8.4

```
If you want to `ssh` into your instance run the following command
```console
$ pharmer ssh node d1-master -k d1
```
### Cluster Scaling

Scaling a cluster refers following meanings:-
 1. Increment the number of nodes of a certain node group
 2. Decrement the number of nodes of a certain node group
 3. Introduce a new node group with a number of nodes
 4. Drop existing node group

To see the current node groups list, you need to run following command:
```console
$ pharmer get nodegroup -k d1
NAME       Cluster   Node      SKU
2gb-pool   d1        2         2gb       
master     d1        1         2gb 
```
* **Updating existing NG**

For scenario 1 & 2 we need to update our existing node group. To update existing node group configuration run
the following command.

```yaml
$ pharmer edit nodegroup 2gb-pool -k d1

# Please edit the object below. Lines beginning with a '#' will be ignored,
# and an empty file will abort the edit. If an error occurs while saving this file will be
# reopened with the relevant failures.
#
apiVersion: v1alpha1
kind: NodeGroup
metadata:
  clusterName: d1
  creationTimestamp: 2017-11-24T05:10:04Z
  labels:
    node-role.kubernetes.io/node: ""
  name: 2gb-pool
  uid: bb4bee00-d0d5-11e7-942e-382c4a73a7c4
spec:
  nodes: 2
  template:
    spec:
      sku: 2gb
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
$ pharmer create ng --nodes=1gb=1 -k d1

$ pharmer get nodegroups -k d1
NAME       Cluster   Node      SKU
1gb-pool   d1        1         VC1M      
2gb-pool   d1        2         2gb       
master     d1        1         2gb 

```
You can see the yaml of newly created node group, you need to run
```yaml
$ pharmer get ng 1gb-pool -k stas -o yaml
apiVersion: v1alpha1
kind: NodeGroup
metadata:
  clusterName: d1
  creationTimestamp: 2017-11-24T06:14:20Z
  labels:
    node-role.kubernetes.io/node: ""
  name: 1gb-pool
  uid: b5afb492-d0de-11e7-b58f-382c4a73a7c4
spec:
  nodes: 1
  template:
    spec:
      sku: 1gb
status:
  nodes: 0

```
* **Delete existing NG**

If you want delete existing node group following command will help.
```yaml
$ pharmer delete ng 2gb-pool -k d1

$ pharmer get ng 2gb-pool -k d1 -o yaml
apiVersion: v1alpha1
kind: NodeGroup
metadata:
  clusterName: d1
  creationTimestamp: 2017-11-24T05:10:04Z
  deletionTimestamp: 2017-11-24T06:15:49Z
  labels:
    node-role.kubernetes.io/node: ""
  name: 2gb-pool
  uid: bb4bee00-d0d5-11e7-942e-382c4a73a7c4
spec:
  nodes: 1
  template:
    spec:
      sku: 2gb
status:
  nodes: 0
```
Here,

 - `metadata.deletionTimestamp`: will appear if node group deleted command was run

After completing your change on the node groups, you need to apply that via `pharmer` so that changes will be applied
on provider cluster.

```console
$ pharmer apply d1
```
This command will take care of your actions that you applied on the node groups recently.

```console

$ pharmer get nodegroups -k d1
NAME       Cluster   Node      SKU
1gb-pool   d1        1         VC1M           
master     d1        1         2gb 

```
### Cluster Upgrading

To upgrade your cluster firstly you need to check if there any update available for your cluster and latest kubernetes version.
To check run:

```console
$ pharmer describe cluster d1
Name:		d1
Version:	v1.8.0
NodeGroup:
  Name       Node
  ----       ------
  1gb-pool    1
  master     1
[upgrade/versions] Cluster version: v1.8.0
[upgrade/versions] kubeadm version: v1.8.4
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

	pharmer edit cluster d1 --kubernetes-version=v1.8.4

_____________________________________________________________________

```
Then, if you decided to upgrade you cluster run the command that are showing on describe command.
```console
$ pharmer edit cluster sd1 --kubernetes-version=v1.8.4
cluster "d1" updated
```
You can verify your changes by checking the yaml of the cluster.
```yaml
$ pharmer get cluster d1 -o yaml
apiVersion: v1alpha1
kind: Cluster
metadata:
  creationTimestamp: 2017-11-10T11:26:23Z
  generation: 1510313183635265805
  name: d1
  uid: ACID-K8S-C-v4ec7ho3if3tt5p
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
    cloudProvider: digitalocean
    instanceImage: ubuntu-16-04-x64
    region: nyc3
    sshKeyName: d1-fpw40n
    zone: nyc3
  credentialName: do
  frontProxyCACertName: front-proxy-ca
  kubernetesVersion: v1.8.4
  networking:
    dnsDomain: cluster.local
    networkProvider: calico
    podSubnet: 192.168.0.0/16
    serviceSubnet: 10.96.0.0/12
status:
  apiServer:
  - address: 10.132.42.222
    type: InternalIP
  - address: 159.203.97.84
    type: ExternalIP
  cloud:
    sshKeyExternalID: "15733879"
  phase: Ready

```
Here, `spec.kubernetesVersion` is changed to `v1.8.4` from `v1.8.0`

If everything looks ok, then run:
```console
$ pharmer apply d1
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
$ pharmer delete cluster d1
```
Then, the yaml file looks like

```yaml
$ pharmer get cluster d1 -o yaml
apiVersion: v1alpha1
kind: Cluster
metadata:
  creationTimestamp: 2017-11-10T11:26:23Z
  deletionTimestamp: 2017-11-10T11:26:23Z
  generation: 1510313183635265805
  name: d1
  uid: ACID-K8S-C-v4ec7ho3if3tt5p
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
    cloudProvider: digitalocean
    instanceImage: ubuntu-16-04-x64
    region: nyc3
    sshKeyName: d1-fpw40n
    zone: nyc3
  credentialName: do
  frontProxyCACertName: front-proxy-ca
  kubernetesVersion: v1.8.4
  networking:
    dnsDomain: cluster.local
    networkProvider: calico
    podSubnet: 192.168.0.0/16
    serviceSubnet: 10.96.0.0/12
status:
  apiServer:
  - address: 10.132.42.222
    type: InternalIP
  - address: 159.203.97.84
    type: ExternalIP
  cloud:
    sshKeyExternalID: "15733879"
  phase: Ready

```
Here,

- `metadata.deletionTimestamp`: is set when cluster deletion command was applied.

Now, to apply delete on provider cluster run
```console
$ pharmer apply d1
```

**Congratulations !!!** , you're an official `pharmer` user now.
