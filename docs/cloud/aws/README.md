---
title: AWS Overview
menu:
  product_pharmer_0.1.0-alpha.1:
    identifier: aws-overview
    name: Overview
    parent: aws
    weight: 10
product_name: pharmer
menu_name: product_pharmer_0.1.0-alpha.1
section_menu_id: cloud
url: /products/pharmer/0.1.0-alpha.1/cloud/aws/
aliases:
  - /products/pharmer/0.1.0-alpha.1/cloud/aws/README/
---

# Running Kubernetes on [AWS](https://aws.amazon.com/)

Following example will use `pharmer ` to create a Kubernetes cluster with 2 worker node servers and a master server (i,e, 3 servers in you cluster).

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

* **Setup IAM User**

In order to create cluster within [AWS](https://aws.amazon.com/), `pharmer` needs a dedicated IAM user. `pharmer` use this user's API credential.

The `pharmer` user needs following permission to works properly.

![pharmer-iam](/docs/images/aws/pharmer-iam.png)

If you have installed [aws cli](http://docs.aws.amazon.com/cli/latest/userguide/installing.html) locally, then you can use the following
command to create `pharmer` IAM user.

```console
$ aws iam create-group --group-name pharmer

$ aws iam attach-group-policy --policy-arn arn:aws:iam::aws:policy/AmazonEC2FullAccess --group-name pharmer
$ aws iam attach-group-policy --policy-arn arn:aws:iam::aws:policy/AmazonRoute53FullAccess --group-name pharmer
$ aws iam attach-group-policy --policy-arn arn:aws:iam::aws:policy/IAMFullAccess --group-name  pharmer
$ aws iam attach-group-policy --policy-arn arn:aws:iam::aws:policy/AmazonVPCFullAccess --group-name pharmer
$ aws iam create-user --user-name pharmer


$ aws iam add-user-to-group --user-name pharmer --group-name pharmer
$ aws iam create-access-key --user-name pharmer

```

Use this access key while importing credentials on pharmer

From command line, run the following command and paste those keys.
```console
$ pharmer create credential aws
```
![aws-credential](/docs/images/aws/aws-credential.png)

Here, `aws` is the credential name, which must be unique within your storage.

To view credential file you can run:
```yaml
$ pharmer get credentials aws -o yaml
apiVersion: v1alpha1
kind: Credential
metadata:
  creationTimestamp: 2017-10-06T04:43:53Z
  name: aws
spec:
  data:
    accessKeyID: <key-id>
    secretAccessKey: <access-key>
  provider: AWS


```
Here,
 - `spec.data.accessKeyID` is the aws access key id
 - `spec.data.secretAccessKey` is the security access key that you provided which can be edited by following command:
```console
$ pharmer edit credential aws
```

To see the all credentials you need to run following command.

```console
$ pharmer get credentials
NAME         Provider       Data
aws          AWS            accessKeyID=AKIAJKUZAD3HM7OEKPNA, secretAccessKey=*****
```

You can also see the stored credential from the following location:
```console
~/.pharmer/store.d/credentials/aws.json
```

 **Cluster IAM User**

 While creating cluster within AWS `pharmer` creates following IAM roles and policies
 * [IAM master policy](https://github.com/pharmer/pharmer/blob/2cd28d23ea7943702729c60bc750a3a97e38b653/cloud/providers/aws/iam.go#L4)
 * [IAM master role](https://github.com/pharmer/pharmer/blob/2cd28d23ea7943702729c60bc750a3a97e38b653/cloud/providers/aws/iam.go#L73)
 * [IAM node policy](https://github.com/pharmer/pharmer/blob/2cd28d23ea7943702729c60bc750a3a97e38b653/cloud/providers/aws/iam.go#L88)
 * [IAM node role](https://github.com/pharmer/pharmer/blob/2cd28d23ea7943702729c60bc750a3a97e38b653/cloud/providers/aws/iam.go#L155)

### Cluster provisioning

There are two steps to create a Kubernetes cluster using `pharmer`.
In first step `pharmer` create basic configuration file with user choice. Then in second step `pharmer` applies those
information to create cluster on specific provider.

Here, we discuss how to use `pharmer` to create a Kubernetes cluster on `aws`
 * **Cluster Creating:** We want to create a cluster with following information:
    - Provider: AWS
    - Cluster name: a1
    - Location: us-west-1a (California)
    - Number of nodes: 2
    - Node sku: t2.medium (cpu:2, ram: 4)
    - Kubernetes version: 1.11.0
    - Credential name: [aws](#credential-importing)

For location code and sku details click [hrere](https://github.com/pharmer/pharmer/blob/master/data/files/aws/cloud.json)

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

$ pharmer create cluster a1 \
	--v=5 \
	--provider=aws \
	--zone=us-west-1a \
	--nodes=t2.medium=2 \
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
        |          |__ id_a1-kbmixz
        |          |
        |          |__ id_a1-kbmixz.pub
        |
        |__ a1.json
```
Here,

   - `/v1/nodegroups/`: contains the node groups information. [Check below](#cluster-scaling) for node group operations.You can see the node group list using following command.
   ```console
$ pharmer get nodegroups -k a1
```
   - `v1/pki`: contains the cluster certificate information containing `ca` and `front-proxy-ca`.
   - `v1/ssh`: has the ssh credentials on cluster's nodes. With this key you can `ssh` into any node on a cluster
   - `v1.json`: contains the cluster resource information

You can view your cluster configuration file by following command.
```yaml
$ pharmer get cluster a1 -o yaml
apiVersion: v1alpha1
kind: Cluster
metadata:
  creationTimestamp: 2017-11-27T05:47:42Z
  generation: 1511761662231600187
  name: a1
  uid: 7c20e9ae-d336-11e7-b746-382c4a73a7c4
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
    aws:
      iamProfileMaster: kubernetes-master
      iamProfileNode: kubernetes-node
      masterIPSuffix: ".9"
      masterSGName: a1-master-horacq
      nodeSGName: a1-node-rnoe2r
      subnetCidr: 172.20.0.0/24
      vpcCIDR: 172.20.0.0/16
      vpcCIDRBase: "172.20"
    cloudProvider: aws
    region: us-west-1
    sshKeyName: a1-kbmlxz
    zone: us-west-1a
  credentialName: aws
  frontProxyCACertName: front-proxy-ca
  kubernetesVersion: v1.11.0
  networking:
    masterSubnet: 10.246.0.0/24
    networkProvider: calico
    nonMasqueradeCIDR: 10.0.0.0/8
status:
  cloud:
    aws: {}
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
$ pharmer edit cluster a1
```

* **Applying:** If everything looks ok, we can now apply the resources. This actually creates resources on `Aws`.
 Up to now we've only been working locally.

 To apply run:
 ```console
$ pharmer apply a1
```
 Now, `pharmer` will apply that configuration, thus create a Kubernetes cluster. After completing task the configuration file of
 the cluster will be look like
```yaml
$ pharmer get cluster a1 -o yaml
apiVersion: v1alpha1
kind: Cluster
metadata:
  creationTimestamp: 2017-11-27T05:47:42Z
  generation: 1511761662231600187
  name: a1
  uid: 7c20e9ae-d336-11e7-b746-382c4a73a7c4
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
    aws:
      iamProfileMaster: kubernetes-master
      iamProfileNode: kubernetes-node
      masterIPSuffix: ".9"
      masterSGName: a1-master-horacq
      nodeSGName: a1-node-rnoe2r
      subnetCidr: 172.20.0.0/24
      vpcCIDR: 172.20.0.0/16
      vpcCIDRBase: "172.20"
    cloudProvider: aws
    instanceImage: ami-73f7da13
    os: ubuntu
    region: us-west-1
    sshKeyName: a1-kbmlxz
    zone: us-west-1a
  credentialName: aws
  frontProxyCACertName: front-proxy-ca
  kubernetesVersion: v1.11.0
  networking:
    masterSubnet: 10.246.0.0/24
    networkProvider: calico
    nonMasqueradeCIDR: 10.0.0.0/8
status:
  apiServer:
  - address: 54.153.7.81
    type: ExternalIP
  - address: 172.20.0.9
    type: InternalIP
  cloud:
    aws:
      dhcpOptionsID: dopt-edaf9389
      igwID: igw-012c8565
      masterSGID: sg-0d5f656b
      nodeSGID: sg-a15f65c7
      routeTableID: rtb-8a9226ed
      subnetID: subnet-5ba66a00
      volumeID: vol-02018cbd914dc98a8
      vpcID: vpc-1de2f079
  phase: Ready
```

Here,

  - `status.phase`: is ready. So, you can use your cluster from local machine.
  - `status.apiserver` is the cluster's apiserver address
  - `status.cloud.aws` contains provider resource information that are created by `pharmer` while creating cluster.

To get the `kubectl` configuration file(kubeconfig) on your local filesystem run the following command.
```console
$ pharmer use cluster a1
```
If you don't have `kubectl` installed click [here](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

Now you can run `kubectl get nodes` and verify that your kubernetes 1.11.0 is running.

```console
$ kubectl get nodes
NAME                                         STATUS    ROLES     AGE       VERSION
ip-172-20-0-236.us-west-1.compute.internal   Ready     node      4m        v1.11.0
ip-172-20-0-9.us-west-1.compute.internal     Ready     master    6m        v1.11.0

```
If you want to `ssh` into your instance run the following command
```console
$ pharmer ssh node ip-172-20-0-9.us-west-1.compute.internal -k a1
```

### Cluster Scaling

Scaling a cluster refers following meanings:-
 1. Increment the number of nodes of a certain node group
 2. Decrement the number of nodes of a certain node group
 3. Introduce a new node group with a number of nodes
 4. Drop existing node group

To see the current node groups list, you need to run following command:
```console
$ pharmer get nodegroup -k a1
NAME             Cluster   Node      SKU
master           a1        1         m3.large
t2.medium-pool   a1        1         t2.medium
```

* **Updating existing NG**

For scenario 1 & 2 we need to update our existing node group. To update existing node group configuration run
the following command.

```yaml
$ pharmer edit nodegroup t2.medium-pool  -k a1

# Please edit the object below. Lines beginning with a '#' will be ignored,
# and an empty file will abort the edit. If an error occurs while saving this file will be
# reopened with the relevant failures.
#
apiVersion: v1alpha1
kind: NodeGroup
metadata:
  clusterName: a1
  creationTimestamp: 2017-11-27T05:47:42Z
  labels:
    node-role.kubernetes.io/node: ""
  name: t2.medium-pool
  uid: 7c66e43e-d336-11e7-b746-382c4a73a7c4
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
$ pharmer create ng --nodes=t2.small=1 -k a1

$ pharmer get nodegroups -k a1
NAME             Cluster   Node      SKU
master           a1        1         m3.large
t2.medium-pool   a1        1         t2.medium
t2.small-pool    a1        1         t2.small

```
- Spot NG:

To know about aws spot instance click [here](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/spot-fleet.html)
To create an spot node group run

```console
$ pharmer create ng --nodes=t2.small=1 --type=spot --spot-price-max=1  -k a1
```

Here,
- `type` = `spot`
- `spot-price-max` is the maximum spot price

You can see the yaml of the newly created node group by running

```yaml
$ pharmer get ng t2.small-pool -k a1 -o yaml
apiVersion: v1alpha1
kind: NodeGroup
metadata:
  clusterName: a1
  creationTimestamp: 2017-11-27T06:23:41Z
  labels:
    node-role.kubernetes.io/node: ""
  name: t2.small-pool
  uid: 82f98e93-d33b-11e7-bf05-382c4a73a7c4
spec:
  nodes: 1
  template:
    spec:
      nodeDiskSize: 100
      nodeDiskType: gp2
      sku: t2.small
      spotPriceMax: 1
      type: spot
status:
  nodes: 0

```

* **Delete existing NG**

If you want delete existing node group following command will help.
```yaml
$ pharmer delete ng t2.small-pool -k a1

$ pharmer get ng t2.small-pool -k a1 -o yaml
apiVersion: v1alpha1
kind: NodeGroup
metadata:
  clusterName: a1
  creationTimestamp: 2017-11-27T06:23:41Z
  deletionTimestamp: 2017-11-27T06:35:43Z
  labels:
    node-role.kubernetes.io/node: ""
  name: t2.small-pool
  uid: 82f98e93-d33b-11e7-bf05-382c4a73a7c4
spec:
  nodes: 1
  template:
    spec:
      nodeDiskSize: 100
      nodeDiskType: gp2
      sku: t2.small
      spotPriceMax: 1
      type: spot
status:
  nodes: 0

```
Here,

 - `metadata.deletionTimestamp`: will appear if node group deleted command was run

After completing your change on the node groups, you need to apply that via `pharmer` so that changes will be applied
on provider cluster.

```console
$ pharmer apply a1
```
This command will take care of your actions that you applied on the node groups recently.

```console
$ pharmer get ng -k a1
NAME             Cluster   Node      SKU
master           a1        1         m3.large
t2.medium-pool   a1        1         t2.medium
```

### Cluster Upgrading

To upgrade your cluster firstly you need to check if there any update available for your cluster and latest kubernetes version.
To check run:
```console
$ pharmer describe cluster a1

Name:		a1
Version:	v1.11.0
NodeGroup:
  Name             Node
  ----             ------
  master           1
  t2.medium-pool   1
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

	pharmer edit cluster a1 --kubernetes-version=v1.11.0

Note: Before you do can perform this upgrade, you have to update kubeadm to v1.11.1

_____________________________________________________________________

```

Then, if you decided to upgrade you cluster run the command that are showing on describe command.
```console
$ pharmer edit cluster a1 --kubernetes-version=v1.11.1
cluster "a1" updated
```

You can verify your changes by checking the yaml of the cluster.
```console
$ pharmer get cluster a1 -o yaml

apiVersion: v1alpha1
kind: Cluster
metadata:
  creationTimestamp: 2017-11-27T05:47:42Z
  generation: 1511765004665530278
  name: a1
  uid: 7c20e9ae-d336-11e7-b746-382c4a73a7c4
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
    aws:
      iamProfileMaster: kubernetes-master
      iamProfileNode: kubernetes-node
      masterIPSuffix: ".9"
      masterSGName: a1-master-horacq
      nodeSGName: a1-node-rnoe2r
      subnetCidr: 172.20.0.0/24
      vpcCIDR: 172.20.0.0/16
      vpcCIDRBase: "172.20"
    cloudProvider: aws
    instanceImage: ami-73f7da13
    os: ubuntu
    region: us-west-1
    sshKeyName: a1-kbmlxz
    zone: us-west-1a
  credentialName: aws
  frontProxyCACertName: front-proxy-ca
  kubernetesVersion: v1.11.1
  networking:
    masterSubnet: 10.246.0.0/24
    networkProvider: calico
    nonMasqueradeCIDR: 10.0.0.0/8
status:
  apiServer:
  - address: 54.153.7.81
    type: ExternalIP
  - address: 172.20.0.9
    type: InternalIP
  cloud:
    aws:
      dhcpOptionsID: dopt-edaf9389
      igwID: igw-012c8565
      masterSGID: sg-0d5f656b
      nodeSGID: sg-a15f65c7
      routeTableID: rtb-8a9226ed
      subnetID: subnet-5ba66a00
      volumeID: vol-02018cbd914dc98a8
      vpcID: vpc-1de2f079
  phase: Ready
```

Here, `spec.kubernetesVersion` is changed to `v1.11.1` from `v1.11.0`

If everything looks ok, then run:
```console
$ pharmer apply s1
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
$ pharmer delete cluster a1
```
Then, the yaml file looks like

```yaml
$ pharmer get cluster a1 -o yaml

apiVersion: v1alpha1
kind: Cluster
metadata:
  creationTimestamp: 2017-11-27T05:47:42Z
  deletionTimestamp: 2017-11-27T06:53:05Z
  generation: 1511765004665530278
  name: a1
  uid: 7c20e9ae-d336-11e7-b746-382c4a73a7c4
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
    aws:
      iamProfileMaster: kubernetes-master
      iamProfileNode: kubernetes-node
      masterIPSuffix: ".9"
      masterSGName: a1-master-horacq
      nodeSGName: a1-node-rnoe2r
      subnetCidr: 172.20.0.0/24
      vpcCIDR: 172.20.0.0/16
      vpcCIDRBase: "172.20"
    cloudProvider: aws
    instanceImage: ami-73f7da13
    os: ubuntu
    region: us-west-1
    sshKeyName: a1-kbmlxz
    zone: us-west-1a
  credentialName: aws
  frontProxyCACertName: front-proxy-ca
  kubernetesVersion: v1.11.1
  networking:
    masterSubnet: 10.246.0.0/24
    networkProvider: calico
    nonMasqueradeCIDR: 10.0.0.0/8
status:
  apiServer:
  - address: 54.153.7.81
    type: ExternalIP
  - address: 172.20.0.9
    type: InternalIP
  cloud:
    aws:
      dhcpOptionsID: dopt-edaf9389
      igwID: igw-012c8565
      masterSGID: sg-0d5f656b
      nodeSGID: sg-a15f65c7
      routeTableID: rtb-8a9226ed
      subnetID: subnet-5ba66a00
      volumeID: vol-02018cbd914dc98a8
      vpcID: vpc-1de2f079
  phase: Deleting
```
Here,

- `metadata.deletionTimestamp`: is set when cluster deletion command was applied.

Now, to apply delete on provider cluster run
```console
$ pharmer apply a1
```

**Congratulations !!!** , you're an official `pharmer` user now.







