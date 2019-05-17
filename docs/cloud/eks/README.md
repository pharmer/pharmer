---
title: Amazon EKS
menu:
  product_pharmer_0.1.0-alpha.1:
    identifier: eks-overview
    name: Overview
    parent: eks
    weight: 30
product_name: pharmer
menu_name: product_pharmer_0.1.0-alpha.1
section_menu_id: cloud
url: /products/pharmer/0.1.0-alpha.1/cloud/eks/
aliases:
  - /products/pharmer/0.1.0-alpha.1/cloud/eks/README/
---

# Running Kubernetes on [Amazon EKS](https://docs.aws.amazon.com/eks/latest/userguide/getting-started.html)

Following example will use `pharmer ` to create a Kubernetes cluster with 1 worker node on Amazon EKS.

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

You also need `Guard` installed on your local machine. To install Guard click [here](https://appscode.com/products/guard/0.1.3/setup/install/#install-guard-as-cli)

`AWS CLI` tool need to be installed. For this follow [Configuring the AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html) in the AWS command Line Interface User Guide.

### Pharmer storage

To store your cluster  and credential resource, `pharmer` use [vfs](/docs/cli/vfs.md) as default storage
provider. There is another provider [postgres database](/docs/cli/xorm.md) available for storing resources.

To know more click [here](/docs/cli/datastore.md)

In this document we will use local file system ([vfs](/docs/cli/vfs.md)) as a storage provider.

### Credential importing

You can use your [aws credential](/docs/cloud/aws/README.md#credential-importing) to creat cluster in Amazon EKS.


### Cluster provisioning

There are two steps to create a Kubernetes cluster using `pharmer`.
In first step `pharmer` create basic configuration file with user choice. Then in second step `pharmer` applies those
information to create cluster on specific provider.

Here, we discuss how to use `pharmer` to create a Kubernetes cluster on `Amazon EKS`
 * **Cluster Creating:** We want to create a cluster with following information:
    - Provider: EKS
    - Cluster name: eksx
    - Location: us-west-2a (Oregon)
    - Number of nodes: 1
    - Node sku: t2.medium
    - Kubernetes version: 1.10
    - Credential name: [aws](/docs/cloud/aws/README.md#credential-importing)

For location code and sku details click [hrere](https://github.com/pharmer/cloud/blob/master/data/json/apis/cloud.pharmer.io/v1/cloudproviders/eks.json)
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
      --stderrthreshold severity         logs at or above this threshold go to stderr
  -v, --v Level                          log level for V logs
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging
 ```

So, we need to run following command to create cluster with our information.

```console

$ pharmer create cluster eksx \
	--provider=eks \
	--zone=us-west-2a \
	--nodes=t2.medium=1 \
	--credential-uid=aws \
	--kubernetes-version=v1.10
```

To know about [pod networks](https://kubernetes.io/docs/concepts/cluster-administration/networking/) supports in `pharmer` click [here](/docs/networking.md)

The directory structure of the storage provider will be look like:

```console
~/.pharmer/store.d/$USER/clusters/
        |-- v1
        |    |__ nodegroups
        |    |       |
        |    |       |__ t2.medium-pool.json
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
        |          |__ id_eksx-unu5i3
        |          |
        |          |__ id_eksx-unu5i3.pub
        |
        |__ eksx.json
```
Here,

   - `/v1/nodegroups/`: contains the node groups information. [Check below](#cluster-scaling) for node group operations.You can see the node group list using following command.
   ```console
$ pharmer get nodegroups -k eksx
```
   - `v1/pki`: contains the cluster certificate information containing `ca` and `front-proxy-ca`.
   - `v1/ssh`: has the ssh credentials on cluster's nodes. With this key you can `ssh` into any node on a cluster
   - `v1.json`: contains the cluster resource information
You can view your cluster configuration file by following command.
```yaml
$ pharmer get cluster eksx -o yaml
apiVersion: v1alpha1
kind: Cluster
metadata:
  creationTimestamp: 2018-06-23T05:14:21Z
  generation: 1529730861349559445
  name: eksx
  uid: 496e8ac4-76a4-11e8-96b1-382c4a73a7c4
spec:
  api:
    advertiseAddress: ""
    bindPort: 0
  caCertName: ca
  cloud:
    cloudProvider: eks
    region: us-west-2
    sshKeyName: eksx-dnafc3
    zone: us-west-2a
  credentialName: aws
  frontProxyCACertName: front-proxy-ca
  kubernetesVersion: "1.10"
  networking:
    networkProvider: calico
status:
  cloud:
    eks: {}
  phase: Pending
```

Here,

* `metadata.name` refers the cluster name, which should be unique within your cluster list.
* `metadata.uid` is a unique ACID, which is generated by pharmer
* `spec.cloud` specifies the cloud provider information.
* `spc.cloud.sshKeyName` shows which ssh key added to cluster instance.
* `spec.kubernetesVersion` is the cluster server version. It can be modified.
* `spec.credentialName` is the credential name which is provider during cluster creation command.
* `status.phase` may be `Pending`, `Ready`, `Deleting`, `Deleted`, `Upgrading` depending on current cluster status.

You can modify this configuration by:
```console
$ pharmer edit cluster eksx
```

* **Applying:** If everything looks ok, we can now apply the resources. This actually creates resources on `Amazon EKS`.
 Up to now we've only been working locally.

 To apply run:
 ```console
$ pharmer apply eksx
```

Now, `pharmer` will apply that configuration, thus create a Kubernetes cluster. After completing task the configuration file of
the cluster will be look like

```yaml
apiVersion: v1alpha1
kind: Cluster
metadata:
  creationTimestamp: 2018-06-23T05:14:21Z
  generation: 1529730861349559445
  name: eksx
  uid: 496e8ac4-76a4-11e8-96b1-382c4a73a7c4
spec:
  api:
    advertiseAddress: ""
    bindPort: 0
  caCertName: ca
  cloud:
    cloudProvider: eks
    instanceImage: ami-73a6e20b
    region: us-west-2
    sshKeyName: eksx-dnafc3
    zone: us-west-2a
  credentialName: aws
  frontProxyCACertName: front-proxy-ca
  kubernetesVersion: "1.10"
  networking:
    networkProvider: calico
status:
  cloud:
    eks:
      roleArn: arn:aws:iam::452618475015:role/EKS-eksx-ServiceRole-AWSServiceRoleForAmazonEKS-VKPC2W53I561
      securityGroup: sg-eca7a69d
      subnetID: subnet-92c6b2eb,subnet-218e1f6a,subnet-4aefac10
      vpcID: vpc-5c908d25
  phase: Ready
```

Here,

  `status.phase`: is ready. So, you can use your cluster from local machine.

To get the `kubectl` configuration file(kubeconfig) on your local filesystem run the following command.
```console
$ pharmer use cluster eksx
```
If you don't have `kubectl` installed click [here](https://kubernetes.io/docs/tasks/tools/install-kubectl/)


Now you can run `kubectl get nodes` and verify that your kubernetes 1.10 is running.


```console
$ kubectl get nodes

NAME                                          STATUS    ROLES     AGE       VERSION
ip-192-168-151-1.us-west-2.compute.internal   Ready     <none>    5m        v1.10.3
```

### Cluster Scaling

Scaling a cluster refers following meanings:-
 1. Increase the number of nodes of a certain node group
 2. Decrease the number of nodes of a certain node group
 3. Introduce a new node group with a number of nodes
 4. Drop existing node group

To see the current node groups list, you need to run following command:
```console
$ pharmer get nodegroup -k eksx
NAME             Cluster   Node      SKU
t2.medium-pool   eksx      1         t2.medium
```

* **Updating existing NG**

For scenario 1 & 2 we need to update our existing node group. To update existing node group configuration run
the following command.

```console
$ pharmer edit nodegroup t2.medium-pool  -k eksx

# Please edit the object below. Lines beginning with a '#' will be ignored,
# and an empty file will abort the edit. If an error occurs while saving this file will be
# reopened with the relevant failures.
#
apiVersion: v1alpha1
kind: NodeGroup
metadata:
  clusterName: eksx
  creationTimestamp: 2018-06-22T11:00:15Z
  labels:
    node-role.kubernetes.io/node: ""
  name: t2.medium-pool
  uid: 714dbad0-760b-11e8-b2c6-382c4a73a7c4
spec:
  nodes: 1
  template:
    spec:
      nodeDiskSize: 100
      nodeDiskType: gp2
      sku: t2.medium
      type: regular
status:
  nodes: 0

```

Here,
* `metadata.name` refers the node group name, which is unique within a cluster.
* `metadata.labels` specifies the label of the nodegroup, which will be add to all nodes of following node group.
    * For node label will be like `"node-role.kubernetes.io/node": ""`
* `metadata.clusterName` indicates the cluster, which has this node group.
* `spec.nodes` shows the number of nodes for this following group.
* `spec.template.sku` refers the size of the machine
* `status.node` shows the number of nodes that are really present on the current cluster while scaling

To update number of nodes for this nodegroup modify the `node` number under `spec` field.

* **Introduce new NG**

To add a new node group for an existing cluster you need to run

```console
$  pharmer create ng --nodes=t2.large=1 -k eksx

$ pharmer get nodegroup -k eksx
NAME             Cluster   Node      SKU
t2.large-pool    eksx      1         t2.large
t2.medium-pool   eksx      1         t2.medium
```

You can see the yaml of newly created node group, you need to run
```yaml
$ pharmer get nodegroup t2.large-pool -o yaml -k eksx
apiVersion: v1alpha1
kind: NodeGroup
metadata:
  clusterName: eksx
  creationTimestamp: 2018-06-22T11:34:33Z
  labels:
    node-role.kubernetes.io/node: ""
  name: t2.large-pool
  uid: 3c4bec6c-7610-11e8-a511-382c4a73a7c4
spec:
  nodes: 1
  template:
    spec:
      nodeDiskSize: 100
      nodeDiskType: gp2
      sku: t2.large
      type: regular
status:
  nodes: 0
```
* **Delete existing NG**

If you want delete existing node group following command will help.

```yaml
$ pharmer delete ng t2.medium-pool -k eksx

$ pharmer get nodegroup t2.medium-pool -k eksx -o yaml
apiVersion: v1alpha1
kind: NodeGroup
metadata:
  clusterName: eksx
  creationTimestamp: 2018-06-22T11:00:15Z
  deletionTimestamp: 2018-06-22T11:36:03Z
  labels:
    node-role.kubernetes.io/node: ""
  name: t2.medium-pool
  uid: 714dbad0-760b-11e8-b2c6-382c4a73a7c4
spec:
  nodes: 1
  template:
    spec:
      nodeDiskSize: 100
      nodeDiskType: gp2
      sku: t2.medium
      type: regular
status:
  nodes: 0
```

Here,

 - `metadata.deletionTimestamp`: will appear if node group deleted command was run

After completing your change on the node groups, you need to apply that via `pharmer` so that changes will be applied
on provider cluster.

```console
$ pharmer apply eksx
```

This command will take care of your actions that you applied on the node groups recently.

```console
 $ pharmer get ng -k eksx
NAME            Cluster   Node      SKU
t2.large-pool   eksx      1         t2.large
```

### Cluster Upgrading

`Pharmer` currently does not support upgrading cluster on `Amazon EKS`. Only supported version is `1.10` .


## Cluster Backup

To get a backup of your cluster run the following command:

```console
$ pharmer backup cluster --cluster eksx --backup-dir=eksx-backup
```
Here,
   `--backup-dir` is the flag for specifying your backup directory where phamer puts the backup file

After finishing task `pharmer` creates a `.tar.gz` file in your backup directory where you find the backup yaml of your cluster


## Cluster Deleting

To delete your cluster run
```console
$ pharmer delete cluster eksx
```
Then, the yaml file looks like

```yaml
$ pharmer get cluster eksx -o yaml
apiVersion: v1alpha1
kind: Cluster
metadata:
  creationTimestamp: 2018-06-23T05:14:21Z
  deletionTimestamp: 2018-06-23T05:39:36Z
  generation: 1529730861349559445
  name: eksx
  uid: 496e8ac4-76a4-11e8-96b1-382c4a73a7c4
spec:
  api:
    advertiseAddress: ""
    bindPort: 0
  caCertName: ca
  cloud:
    cloudProvider: eks
    instanceImage: ami-73a6e20b
    region: us-west-2
    sshKeyName: eksx-dnafc3
    zone: us-west-2a
  credentialName: aws
  frontProxyCACertName: front-proxy-ca
  kubernetesVersion: "1.10"
  networking:
    networkProvider: calico
status:
  cloud:
    eks:
      roleArn: arn:aws:iam::452618475015:role/EKS-eksx-ServiceRole-AWSServiceRoleForAmazonEKS-VKPC2W53I561
      securityGroup: sg-eca7a69d
      subnetID: subnet-92c6b2eb,subnet-218e1f6a,subnet-4aefac10
      vpcID: vpc-5c908d25
  phase: Deleting

```
Here,

- `metadata.deletionTimestamp`: is set when cluster deletion command was applied.

Now, to apply delete on provider cluster run
```console
$ pharmer apply eksx
```

**Congratulations !!!** , you're an official `pharmer` user now.

