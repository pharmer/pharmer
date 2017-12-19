
---
title: Azure Overview
menu:
  product_pharmer_0.1.0-alpha.1:
    identifier: azure-overview
    name: Overview
    parent: azure
    weight: 10
product_name: pharmer
menu_name: product_pharmer_0.1.0-alpha.1
section_menu_id: cloud
url: /products/pharmer/0.1.0-alpha.1/cloud/azure/
aliases:
  - /products/pharmer/0.1.0-alpha.1/cloud/azure/README/
---

# Running Kubernetes on [Azure](https://azure.microsoft.com/)

Following example will use `pharmer ` to create a Kubernetes cluster with 1 worker node server and a master server (i,e, 2 servers in you cluster).

### Before you start

As a prerequisite, you need to have `pharmer` installed.  To install `pharmer` run the following command.
```console
$ go get -u github.com/pharmer/pharmer
```

### Pharmer storage

To store your cluster  and credential resource, `pharmer` use [vfs](/docs/cli/vfs.md) as default storage
provider. There is another provider [postgres database](/docs/cli/xorm.md) available for storing resources.

To know more click [here](/docs/cli/datastore.md)

In this document we will use local file system ([vfs](/docs/cli/vfs.md)) as a storage provider.

### Credential importing

**Tenant ID:**
From the Portal, if you click on the Help icon in the upper right and then choose `Show Diagnostics` you can find the tenant id in the diagnostic JSON.

You can also find TenantID from the endpoints URL

![azure-api-key](/docs/images/azure/azure-api-key.png)

From command line, run the following command and paste the api key.
```console
$ pharmer create credential azur --issue
```
![azure-credential](/docs/images/azure/azure-credential.png)

Here, `azure` is the credential name, which must be unique within your storage. With `issue` flag you can issue new credential.
If you want to use your existing credential then no need to pass `issue` flag.

To view credential file you can run:

```yaml
$ pharmer get credential azur -o yaml
apiVersion: v1alpha1
kind: Credential
metadata:
  creationTimestamp: null
  name: azur
spec:
  data:
    clientID: <client id>
    clientSecret: <client secret>
    subscriptionID: <subscription id>
    tenantID: <tenant id>
  provider: Azure

```

Here,
 - `spec.data.clientID` is the azure client id
 - `spec.data.clientSecret` is the secret
 - `spec.data.subscriptionID`  is the subscription id of azure account
 - `spec.data.tenantID` is tenant id that you provided which can be edited by following command:
```console
$ phrmer edit credential azur
```

To see the all credentials you need to run following command.

```console
$ pharmer get credentials
NAME         Provider       Data
azur         Azure          tenantID=77226, subscriptionID=1bfc, clientID=bfd2fee, clientSecret=*****
```
You can also see the stored credential from the following location:
```console
~/.pharmer/store.d/credentials/azur.json
```

You can find other credential operations [here](/docs/credential.md)


### Cluster provisioning

There are two steps to create a Kubernetes cluster using `pharmer`.
In first step `pharmer` create basic configuration file with user choice. Then in second step `pharmer` applies those
information to create cluster on specific provider.

Here, we discuss how to use `pharmer` to create a Kubernetes cluster on `azure`
 * **Cluster Creating:** We want to create a cluster with following information:
    - Provider: Azure
    - Cluster name: az1
    - Location: eastus2 (Virginia)
    - Number of nodes: 1
    - Node sku: Standard_D1_v2 (cpu: 1, ram: 3.5, disk: 50)
    - Kubernetes version: 1.8.0
    - Credential name: [azur](#credential-importing)

For location code and sku details click [hrere](https://github.com/pharmer/pharmer/blob/master/data/files/azure/cloud.json)
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
      --env string                       Environment used to enable debugging (default "prod")
      --log_backtrace_at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                   If non-empty, write log files in this directory
      --logtostderr                      log to standard error instead of files (default true)
      --stderrthreshold severity         logs at or above this threshold go to stderr (default 2)
  -v, --v Level                          log level for V logs
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging
```

So, we need to run following command to create cluster with our information.

```console
$ pharmer create cluster az1 \
	--v=5 \
	--provider=azure \
	--zone=westus2 \
	--nodes=Standard_D1_v2=1 \
	--credential-uid=azur \
	--kubernetes-version=v1.8.0

```

To know about [pod networks](https://kubernetes.io/docs/concepts/cluster-administration/networking/) supports in `pharmer` click [here](/docs/networking.md)

The directory structure of the storage provider will be look like:

```console
~/.pharmer/store.d/clusters/
        |-- v1
        |    |__ nodegroups
        |    |       |__ master.json
        |    |       |
        |    |       |__ Standard-D1-v2-pool.json
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
        |          |__ id_az1-5efv4x
        |          |
        |          |__ id_az1-5efv4x.pub
        |
        |__ az1.json
```
Here,

   - `/v1/nodegroups/`: contains the node groups information. [Check below](#cluster-scaling) for node group operations.You can see the node group list using following command.
   ```console
$ pharmer get nodegroups -k az1
```
   - `v1/pki`: contains the cluster certificate information containing `ca` and `front-proxy-ca`.
   - `v1/ssh`: has the ssh credentials on cluster's nodes. With this key you can `ssh` into any node on a cluster
   - `v1.json`: contains the cluster resource information
You can view your cluster configuration file by following command.

```yaml
$ pharmer get cluster az1 -o yaml
apiVersion: v1alpha1
kind: Cluster
metadata:
  creationTimestamp: 2017-12-07T09:47:46Z
  generation: 1512640066640731326
  name: az1
  uid: adf4d166-db33-11e7-a690-382c4a73a7c4
spec:
  api:
    advertiseAddress: ""
    bindPort: 6443
  apiServerExtraArgs:
    cloud-config: /etc/kubernetes/ccm/cloud-config
    kubelet-preferred-address-types: InternalDNS,InternalIP,ExternalDNS,ExternalIP
  authorizationModes:
  - Node
  - RBAC
  caCertName: ca
  cloud:
    azure:
      azureStorageAccountName: k8saz1bofhqq
      resourceGroup: az1
      rootPassword: S105Bf0S1ZmRkutL
      routeTableName: az1-rt
      securityGroupName: az1-nsg
      subnetCidr: 10.240.0.0/16
      subnetName: az1-subnet
      vnetName: az1-vnet
    ccmCredentialName: azure
    cloudProvider: azure
    region: westus2
    sshKeyName: az1-5efv4x
    zone: westus2
  controllerManagerExtraArgs:
    cloud-config: /etc/kubernetes/ccm/cloud-config
  credentialName: azure
  frontProxyCACertName: front-proxy-ca
  kubernetesVersion: v1.8.0
  networking:
    networkProvider: calico
    nonMasqueradeCIDR: 10.0.0.0/8
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
$ pharmer edit cluster az1
```


 **Applying:** If everything looks ok, we can now apply the resources. This actually creates resources on `Scaleway`.
 Up to now we've only been working locally.

 To apply run:
 ```console
$ pharmer apply az1
```

Now, `pharmer` will apply that configuration, thus create a Kubernetes cluster. After completing task the configuration file of
 the cluster will be look like
 ```yaml
 $ pharmer get cluster az1 -o yaml
 apiVersion: v1alpha1
kind: Cluster
metadata:
  creationTimestamp: 2017-12-07T09:47:46Z
  generation: 1512640066640731326
  name: az1
  uid: adf4d166-db33-11e7-a690-382c4a73a7c4
spec:
  api:
    advertiseAddress: ""
    bindPort: 6443
  apiServerExtraArgs:
    cloud-config: /etc/kubernetes/ccm/cloud-config
    kubelet-preferred-address-types: InternalDNS,InternalIP,ExternalDNS,ExternalIP
  authorizationModes:
  - Node
  - RBAC
  caCertName: ca
  cloud:
    azure:
      azureStorageAccountName: k8saz1bofhqq
      resourceGroup: az1
      rootPassword: S105Bf0S1ZmRkutL
      routeTableName: az1-rt
      securityGroupName: az1-nsg
      subnetCidr: 10.240.0.0/16
      subnetName: az1-subnet
      vnetName: az1-vnet
    ccmCredentialName: azure
    cloudProvider: azure
    region: westus2
    sshKeyName: az1-5efv4x
    zone: westus2
  controllerManagerExtraArgs:
    cloud-config: /etc/kubernetes/ccm/cloud-config
  credentialName: azure
  frontProxyCACertName: front-proxy-ca
  kubernetesVersion: v1.8.0
  networking:
    networkProvider: calico
    nonMasqueradeCIDR: 10.0.0.0/8
status:
  apiServer:
  - address: 10.99.81.129
    type: InternalIP
  - address: 147.75.74.213
    type: ExternalIP
  cloud: {}
  phase: Ready
  reservedIP:
  - ip: 52.183.45.79
 ```

Here,

  `status.phase`: is ready. So, you can use your cluster from local machine.

To get the `kubectl` configuration file(kubeconfig) on your local filesystem run the following command.
```console
$ pharmer use cluster az1
```
If you don't have `kubectl` installed click [here](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

Now you can run `kubectl get nodes` and verify that your kubernetes 1.8.0 is running.

```console
$ kubectl get nodes
NAME                        STATUS    AGE       VERSION
standard-d1-v2-pool-z5t4gh   Ready     2m        v1.8.4
az1-master                   Ready     10m       v1.8.4
```

If you want to `ssh` into your instance run the following command
```console
$ pharmer ssh node az1-master  -k az1
```

### Cluster Scaling

Scaling a cluster refers following meanings:-
 1. Increment the number of nodes of a certain node group
 2. Decrement the number of nodes of a certain node group
 3. Introduce a new node group with a number of nodes
 4. Drop existing node group

To see the current node groups list, you need to run following command:

```console
$ pharmer get nodegroups -k az1
NAME               Cluster   Node      SKU
Standard-D1-v2-pool   az1       1         Standard_D1_v2
master                az1       1         Standard_D2_v2
```

* **Updating existing NG**

For scenario 1 & 2 we need to update our existing node group. To update existing node group configuration run
the following command.

```yaml
$ pharmer edit nodegroup  Standard-D1-v2-pool -k az1

# Please edit the object below. Lines beginning with a '#' will be ignored,
# and an empty file will abort the edit. If an error occurs while saving this file will be
# reopened with the relevant failures.
#
apiVersion: v1alpha1
kind: NodeGroup
metadata:
  clusterName: az1
  creationTimestamp: 2017-12-07T09:25:08Z
  labels:
    node-role.kubernetes.io/node: ""
  name: Standard-D1-v2-pool
  uid: 84ae51a2-db30-11e7-933a-382c4a73a7c4
spec:
  nodes: 1
  template:
    spec:
      sku: Standard_D1_v2
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

* **Introducing a new NG**

To add a new node group for an existing cluster you need to run
```console
$ pharmer create ng --nodes=Standard_D2_v2=1 -k az1
```

You can see the yaml of newly created node group, for that you need to run
```yaml
$ pharmer get ng Standard-D2-v2-pool -k az1 -o yaml
apiVersion: v1alpha1
kind: NodeGroup
metadata:
  clusterName: az1
  creationTimestamp: 2017-12-07T10:20:51Z
  labels:
    node-role.kubernetes.io/node: ""
  name: Standard-D2-v2-pool
  uid: 4cca44e3-db38-11e7-809e-382c4a73a7c4
spec:
  nodes: 1
  template:
    spec:
      sku: Standard_D2_v2
      type: regular
status:
  nodes: 0
```

Here,
 - `spec.template.spec.type` = `regular`, for regular type nodes
 - `spec.template.spec.spotPriceMax` is the maximum price of a node


## Cluster Backup

To get a backup of your cluster run the following command:

```console
$ pharmer backup cluster --cluster az1 --backup-dir=az1-backup
```
Here,
   `--backup-dir` is the flag for specifying your backup directory where phamer puts the backup file

After finishing task `pharmer` creates a `.tar.gz` file in your backup directory where you find the backup yaml of your cluster


* **Delete existing NG**

If you want delete existing node group following command will help.
```yaml
$ pharmer delete ng Standard-D1-v2-pool -k az1

$ pharmer get ng Standard-D1-v2-pool -k az1 -o yaml
apiVersion: v1alpha1
kind: NodeGroup
metadata:
  clusterName: az1
  creationTimestamp: 2017-12-07T09:25:08Z
  deletionTimestamp: 2017-12-07T10:22:25Z
  labels:
    node-role.kubernetes.io/node: ""
  name: Standard-D1-v2-pool
  uid: 84ae51a2-db30-11e7-933a-382c4a73a7c4
spec:
  nodes: 1
  template:
    spec:
      sku: Standard_D1_v2
      type: regular
status:
  nodes: 0


```
Here,

 - `metadata.deletionTimestamp`: will appear if node group deleted command was run

After completing your change on the node groups, you need to apply that via `pharmer` so that changes will be applied
on provider cluster.

```console
$ pharmer apply az1
```
This command will take care of your actions that you applied on the node groups recently.

```console
 $ pharmer get nodegroups -k az1
NAME               Cluster   Node      SKU
Standard-D2-v2-pool   az1       1         Standard_D2_v2
master                az1       1         Standard_D2_v2
```

### Cluster Upgrading

To upgrade your cluster firstly you need to check if there any update available for your cluster and latest kubernetes version.
To check run:
```console
$ pharmer describe cluster az1
Name:		az1
Version:	v1.8.0
NodeGroup:
  Name                 Node
  ----                 ------
  Standard-D2-v2-pool   1
  master                1
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

	pharmer edit cluster az1 --kubernetes-version=v1.8.4

```

Then, if you decided to upgrade you cluster run the command that are showing on describe command.
```console
$ pharmer edit cluster az1 --kubernetes-version=v1.8.4
cluster "az1" updated
```
You can verify your changes by checking the yaml of the cluster.
```yaml
$ pharmer get cluster az1 -o yaml
 apiVersion: v1alpha1
kind: Cluster
metadata:
  creationTimestamp: 2017-12-07T09:47:46Z
  generation: 1512640066640731326
  name: az1
  uid: adf4d166-db33-11e7-a690-382c4a73a7c4
spec:
  api:
    advertiseAddress: ""
    bindPort: 6443
  apiServerExtraArgs:
    cloud-config: /etc/kubernetes/ccm/cloud-config
    kubelet-preferred-address-types: InternalDNS,InternalIP,ExternalDNS,ExternalIP
  authorizationModes:
  - Node
  - RBAC
  caCertName: ca
  cloud:
    azure:
      azureStorageAccountName: k8saz1bofhqq
      resourceGroup: az1
      rootPassword: S105Bf0S1ZmRkutL
      routeTableName: az1-rt
      securityGroupName: az1-nsg
      subnetCidr: 10.240.0.0/16
      subnetName: az1-subnet
      vnetName: az1-vnet
    ccmCredentialName: azure
    cloudProvider: azure
    region: westus2
    sshKeyName: az1-5efv4x
    zone: westus2
  controllerManagerExtraArgs:
    cloud-config: /etc/kubernetes/ccm/cloud-config
  credentialName: azure
  frontProxyCACertName: front-proxy-ca
  kubernetesVersion: v1.8.4
  networking:
    networkProvider: calico
    nonMasqueradeCIDR: 10.0.0.0/8
status:
  apiServer:
  - address: 10.99.81.129
    type: InternalIP
  - address: 147.75.74.213
    type: ExternalIP
  cloud: {}
  phase: Ready
  reservedIP:
  - ip: 52.183.45.79
```
Here, `spec.kubernetesVersion` is changed to `v1.8.4` from `v1.8.0`

If everything looks ok, then run:
```console
$ pharmer apply az1
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
$ pharmer delete cluster az1
```

Then, the yaml file looks like
```yaml
$ pharmer get cluster az1 -o yaml
pharmer get cluster az1 -o yaml
 apiVersion: v1alpha1
kind: Cluster
metadata:
  creationTimestamp: 2017-12-07T09:47:46Z
  deletionTimestamp: 2017-12-07T09:48:42Z
  generation: 1512640066640731326
  name: az1
  uid: adf4d166-db33-11e7-a690-382c4a73a7c4
spec:
  api:
    advertiseAddress: ""
    bindPort: 6443
  apiServerExtraArgs:
    cloud-config: /etc/kubernetes/ccm/cloud-config
    kubelet-preferred-address-types: InternalDNS,InternalIP,ExternalDNS,ExternalIP
  authorizationModes:
  - Node
  - RBAC
  caCertName: ca
  cloud:
    azure:
      azureStorageAccountName: k8saz1bofhqq
      resourceGroup: az1
      rootPassword: S105Bf0S1ZmRkutL
      routeTableName: az1-rt
      securityGroupName: az1-nsg
      subnetCidr: 10.240.0.0/16
      subnetName: az1-subnet
      vnetName: az1-vnet
    ccmCredentialName: azure
    cloudProvider: azure
    region: westus2
    sshKeyName: az1-5efv4x
    zone: westus2
  controllerManagerExtraArgs:
    cloud-config: /etc/kubernetes/ccm/cloud-config
  credentialName: azure
  frontProxyCACertName: front-proxy-ca
  kubernetesVersion: v1.8.0
  networking:
    networkProvider: calico
    nonMasqueradeCIDR: 10.0.0.0/8
status:
  apiServer:
  - address: 10.99.81.129
    type: InternalIP
  - address: 147.75.74.213
    type: ExternalIP
  cloud: {}
  phase: Ready
  reservedIP:
  - ip: 52.183.45.79
```

Here,

- `metadata.deletionTimestamp`: is set when cluster deletion command was applied.

Now, to apply delete on provider cluster run
```console
$ pharmer apply az1
```

**Congratulations !!!** , you're an official `pharmer` user now.

