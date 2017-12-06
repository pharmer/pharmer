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
$ go get github.com/pharmer/pharmer
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
 - `spec.data.projectID` is the packet project id
 - `spec.data.apiKey` is the access token that you provided which can be edited by following command:
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

For location code and sku details click [hrere](https://github.com/pharmer/pharmer/blob/master/data/files/packet/cloud.json)
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
        |          |__ id_p1-osfzqn
        |          |
        |          |__ id_p1-osfzqn.pub
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
  creationTimestamp: 2017-12-06T11:57:42Z
  generation: 1512561462750169126
  name: az1
  uid: aa62f647-da7c-11e7-bfaa-382c4a73a7c4
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
      azureStorageAccountName: k8saz1v6iwud
      resourceGroup: az1
      rootPassword: eb-QDZ9POjRg0dhE
      routeTableName: az1-rt
      securityGroupName: az1-nsg
      subnetCidr: 10.240.0.0/16
      subnetName: az1-subnet
      vnetName: az1-vnet
    ccmCredentialName: azure
    cloudProvider: azure
    region: westus2
    sshKeyName: az1-b3kfmp
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
