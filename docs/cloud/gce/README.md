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

Following example will use `pharmer ` to create a Kubernetes cluster with 1 worker nodes and 3 master nodes (i,e, 4 nodes in you cluster).

### Before you start

As a prerequisite, you need to have `pharmer` installed.  To install `pharmer` run the following command.

```console
mkdir -p $(go env GOPATH)/src/github.com/pharmer
cd $(go env GOPATH)/src/github.com/pharmer
git clone https://github.com/pharmer/pharmer
cd pharmer
./hack/make.py

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
  provider: gce

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
~/.pharmer/store.d/$USER/credentials/gce.json
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
    - Kubernetes version: 1.13.5
    - Credential name: [gce](#credential-importing)

For location code and sku details click [hrere](https://github.com/pharmer/cloud/blob/master/data/json/apis/cloud.pharmer.io/v1/cloudproviders/gce.json)

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
$ pharmer create cluster g1 \
	-- masters 3 \
	--provider=gce \
	--zone=us-central1-f \
	--nodes=n1-standard-2=1 \
	--credential-uid=gce \
	--kubernetes-version=v1.13.5
```

To know about [pod networks](https://kubernetes.io/docs/concepts/cluster-administration/networking/) supports in `pharmer` click [here](/docs/networking.md)

The directory structure of the storage provider will be look like:

```console
~/.pharmer/store.d/$USER/clusters/
/home/<user>/.pharmer/store.d/<user>/clusters/
├── g1
│   ├── machine
│   │   ├── g1-master-0.json
│   │   ├── g1-master-1.json
│   │   └── g1-master-2.json
│   ├── machineset
│   │   └── n1-standard-2-pool.json
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
│       ├── id_g1-sshkey
│       └── id_g1-sshkey.pub
└── g1.json

6 directories, 15 files
```
Here,

 - `machine`: conntains information about the master machines to be deployed
  - `machineset`: contains information about the machinesets to be deployed
  - `pki`: contains the cluster certificate information containing `ca`, `front-proxy-ca`, `etcd/ca` and service account keys `sa`
  - `ssh`: has the ssh credentials on cluster's nodes. With this key you can `ssh` into any node on a cluster
  - `g1.json`: contains the cluster resource information 

You can view your cluster configuration file by following command.
```yaml
$ pharmer get cluster g1 -o yaml
kind: Cluster
apiVersion: cluster.pharmer.io/v1beta1
metadata:
  name: g1
  uid: dc86e18d-7853-11e9-8b47-e0d55ee85d14
  generation: 1558063718895666700
  creationTimestamp: '2019-05-17T03:28:38Z'
spec:
  clusterApi:
    kind: Cluster
    apiVersion: cluster.k8s.io/v1alpha1
    metadata:
      name: g1
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
          kind: GCEMachineProviderSpec
          apiVersion: gceproviderconfig/v1alpha1
          metadata:
            creationTimestamp: 
          roles:
          - Master
          zone: us-central1-f
          machineType: ''
          os: ubuntu-1604-xenial-v20170721
          disks:
          - initializeParams:
              diskSizeGb: 30
              diskType: pd-standard
    status: {}
  config:
    masterCount: 3
    cloud:
      cloudProvider: gce
      region: us-central1
      zone: us-central1-f
      instanceImage: ubuntu-1604-xenial-v20170721
      os: ubuntu-1604-lts
      instanceImageProject: ubuntu-os-cloud
      networkProvider: calico
      ccmCredentialName: gce
      sshKeyName: g1-sshkey
      gce:
        NetworkName: default
        NodeTags:
        - g1-node
    kubernetesVersion: v1.13.5
    caCertName: ca
    frontProxyCACertName: front-proxy-ca
    credentialName: gce
    apiServerExtraArgs:
      cloud-config: "/etc/kubernetes/ccm/cloud-config"
      cloud-provider: gce
      kubelet-preferred-address-types: ExternalDNS,ExternalIP,Hostname,InternalDNS,InternalIP
    controllerManagerExtraArgs:
      cloud-config: "/etc/kubernetes/ccm/cloud-config"
      cloud-provider: gce
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
---
kind: Cluster
apiVersion: cluster.pharmer.io/v1beta1
metadata:
  name: g1
  uid: dc86e18d-7853-11e9-8b47-e0d55ee85d14
  generation: 1558063718895666700
  creationTimestamp: '2019-05-17T03:28:38Z'
spec:
  clusterApi:
    kind: Cluster
    apiVersion: cluster.k8s.io/v1alpha1
    metadata:
      name: g1
      namespace: default
      creationTimestamp: 
      annotations:
        gce.clusterapi.k8s.io/firewallg1-allow-api-public: 'true'
        gce.clusterapi.k8s.io/firewallg1-allow-cluster-internal: 'true'
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
          kind: GCEMachineProviderSpec
          apiVersion: gceproviderconfig/v1alpha1
          metadata:
            creationTimestamp: 
          project: ackube
          caKeyPair:
            cert: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUN1RENDQWFDZ0F3SUJBZ0lCQURBTkJna3Foa2lHOXcwQkFRc0ZBREFOTVFzd0NRWURWUVFERXdKallUQWUKRncweE9UQTFNVGN3TXpJNE16bGFGdzB5T1RBMU1UUXdNekk0TXpsYU1BMHhDekFKQmdOVkJBTVRBbU5oTUlJQgpJakFOQmdrcWhraUc5dzBCQVFFRkFBT0NBUThBTUlJQkNnS0NBUUVBdVZ2RE1SRjNRdlY4UDhyQzAxRkFNT095CkQxZnpiUWVGRWpSaUUzVytoeVdEUzRMK2xGVkNxOXVJcEhRRnozOUdvMmdCTzk0L3FaSkM0ZkMyWFY5UGRZckgKSFY3MDlnUFNrNmZTSjh0Zk54bFBISHRmbDFTNTcyZk51anF3ank4NXpsTXhUeXFpUWtTZFhReTkrclBZdmZWVApkVDNvY293YjJnNHVzN3dMaEt3bkt5c2l0V0dvUlFZRkJ3SC95SlZPUnhEdG5BZnZLVEIxcnpPdktrWUJTSjA1CitZK0R1OEdqWjFJa053VTJwWjE4aGw0eVNya0Q4VmpnYktmYUtkdy9UZno4QzBJbEZnY2U0SUhRcWVtcW02UmwKTytMRHdHK3JQeHBIQUFLRVk0aHlBOGhaV3l0QVpyZHFHY1lRT0RxUW9hZDJYQlFGSkJSN1pDQm1XZEFjc1FJRApBUUFCb3lNd0lUQU9CZ05WSFE4QkFmOEVCQU1DQXFRd0R3WURWUjBUQVFIL0JBVXdBd0VCL3pBTkJna3Foa2lHCjl3MEJBUXNGQUFPQ0FRRUFhU1ZZQ0RlWHgvSGJGWWtWaldhYzYvNmZ5cGVnWmYvOGx5b1U0NjI2UGR4eUlDQWcKWmF1QXhhYUhHVEl4NjRGeHQyZUlscGhSejNFRm01Z3hTc1lMV1E0eFRuQ3dkWGRTeGxGdFgwRUEybi9PczlaRwpHeUtNSW1KNjR6UldVeWhxWXRudFRkQkZDS1gwbnZ3NUdScEhWL2kyeGJGQWpvZGEzWnRWN1lBcVhDRHQvcE9mCnpxbS9LOHVCMnJPeFlzeVhMZEpEWGkwbStRQ1AxN0VRMWk1YkFjOEJ4THRidzA2eXBzTDdJVzJMeWVnVFZMaGgKQnJER0h4eWlCZmhOTXhUM2V6SVpYZ05GZHVrYlBydkZ3L0pSTnBNeEFTd1hKRFE1QmxrYkJTY01VdXJ3WFdNKwpSVFhES3NZYVowQ2o4U1NFUEdGbjhXbWViODN2TURhOXZhVjJxQT09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
            key: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb3dJQkFBS0NBUUVBdVZ2RE1SRjNRdlY4UDhyQzAxRkFNT095RDFmemJRZUZFalJpRTNXK2h5V0RTNEwrCmxGVkNxOXVJcEhRRnozOUdvMmdCTzk0L3FaSkM0ZkMyWFY5UGRZckhIVjcwOWdQU2s2ZlNKOHRmTnhsUEhIdGYKbDFTNTcyZk51anF3ank4NXpsTXhUeXFpUWtTZFhReTkrclBZdmZWVGRUM29jb3diMmc0dXM3d0xoS3duS3lzaQp0V0dvUlFZRkJ3SC95SlZPUnhEdG5BZnZLVEIxcnpPdktrWUJTSjA1K1krRHU4R2paMUlrTndVMnBaMThobDR5ClNya0Q4VmpnYktmYUtkdy9UZno4QzBJbEZnY2U0SUhRcWVtcW02UmxPK0xEd0crclB4cEhBQUtFWTRoeUE4aFoKV3l0QVpyZHFHY1lRT0RxUW9hZDJYQlFGSkJSN1pDQm1XZEFjc1FJREFRQUJBb0lCQUhxTUtadktZV0FhcnovNQp6UjhySzlabTQvVnkvNVRKRVBpOU8wNkVYU2c2Ni9oRHJnN2g0OGQ5eUhSNTVOR1A0MkxydzAwU25tSjlPY3pwCmVaaDRDQys5UmZHc015Wm4xcFFhc3ozdUhwQnVJampCZEt5M3JvcVN4WmhuYnczcmVqdG9FMFMwK2p2MzQxWisKc3lnS09iVTFlaTBjZGc4dGhNaUE4ZTJRMk1pb2JOVDdKVlRxaURIQkRtbWpTell5WW5Kc2h0YnRvSGZvWlhwTAo5YTBOMDVzbVNDRVRobFdISWUwVG4zSE0xRDY4VEQ4bEZEY09LdXlDazB3WFJkQ3FFSDQ5RFpzMFRzM2dDK2JoCnpqV2JpcHFsb0d1RkhmTWtsc0Rnc3lyTkhRM21CaVJ5c2sxU1VtTHp1NWxBemFLemVvWVMyTTRXQi9kRjE0UlcKb2w2cjVQRUNnWUVBM1ZrTll2RUR3dGJjbUNvL25hM25oU3JNaG5UQXZ4WHg1a2t3R3pISlNGeWhlU0QxMXNTVAp0MEp1SXlIMzU0cmZlNWRKeEtvT0NmRjMvV1pzaldCQWYyaGZ6YUZRcXoyTmNySzVMWDhhK1BNdGxzUGFHM2JoCkxRU2drY29tY0Ira1I0N2ZzdXZibFVlUjByZ1l2ZGw1anpuK21rWjJmME5ibTNFQmVXWWpvUDBDZ1lFQTFtQmQKT2RCb25uemxSYSt5MVdDNVU1b0pkMXp5TitzUUhPUlJsRTZ1SDU2bWl0cWxoL2sxaGhKRk9OR0h6NkxGZEQvSQo5eXZ4Q3dMcGtUVmFUUi9xMUw3WE1oWlhjT2xFZTB0TUhxaml2dFVaakw2SXdJaU1LL2J5YlBZQVZVODJDV1FoClZlOHhPdzdCcmRQRGZBQzVFYi8zOFZyS1hGRXhnV1g3bTR1MFFzVUNnWUJSU1Btc2d2dXhtbnZwK1dIaFF0TEoKeVh6UVI2SGN5bTlKOVVpUVJBazU1S0o3dkFucnM4YlhQckw1ZmVqdkE4V3NPbE9od0IxbHMySXdFV1A5eXdJRQpoOHplMDhXdkRPeWIyVnc5Zy9iZ3cxVFRqOXJSeVNkS0ErLy9lZkFCcnUwQ1JrcUtCeWxkT2Fvb2F1alRGMEVYCnd1Rm53RWFWMTZPVmdydGEzSkpxOVFLQmdFcXFZWTREWW96ZzMxSDRNZ2RUbXZqZFM3TEJNclA3TVM5KzdsTUQKWEc0eTZicXZFTHhkTmlFdU4rSGtTTE11OUNyYkZIblNXakFGb2FncnR2bnB4ZmEzU1dodWs2SUYvUTRjV2JUTQpDYjJCcDFaMy9sVmd1Y0dPVHoxWUtTR05aenE2SDBvNDl5S2tyeHlHQnk0bmFrNGVXSk05bGdHMVhkSzkzSForCm9CZ3BBb0dCQUtVblZaR09pbU1FbEs3WHhQQzB0NlYvbmRRd1pKdEpWNkVaQWx5M3ErQitoOTZRbzF4d0srbmcKaitVaWt4bmlmVUlFUTF0L2NWMG9vY29CWmpaTlRBeGlsTzJ5eE84QUxsUVh4dm02ZUlybGpRbjBCWkZFbE9mOQpZT2tQZC9SVUROL1FKWktNeHF1UkdhNXNZN2FvdGdaMVJFSEpzVFdRdW80dE5ReEpBb01HCi0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg==
          etcdCAKeyPair:
            cert: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUN5RENDQWJDZ0F3SUJBZ0lCQURBTkJna3Foa2lHOXcwQkFRc0ZBREFWTVJNd0VRWURWUVFERXdwcmRXSmwKY201bGRHVnpNQjRYRFRFNU1EVXhOekF6TWpnME1Gb1hEVEk1TURVeE5EQXpNamcwTUZvd0ZURVRNQkVHQTFVRQpBeE1LYTNWaVpYSnVaWFJsY3pDQ0FTSXdEUVlKS29aSWh2Y05BUUVCQlFBRGdnRVBBRENDQVFvQ2dnRUJBTk9jClZnL0E5eTRONUNUZEdrSjFVSE0xa1VOY1dLRHVaQTlpOUZ3TVRtaUtpZkNtSERab28zUEczVEVEd0xCUU5CbkoKUmM1SnhqTzg1RERwbjlGZFhQSnBZSkFIS0xDK0tyY2F6VURZV2dqaTFDV0RBTFhjc2x0OFRMdWNxL29kamhRTQo4Q0lONE5FcHBiTS9lOXlvZU13M0xaV1FUT015Y3lMVnh3Rm8yWk9xRm1XQlBRaGlaSjI5QjNQNytKTmg2NXhaCmxCQTRsWW9ybXY5RUh6eDcwRkduU1lGeVMwUmI4N3JVejcyTjczRkNDdmRhUi9xL3dBeDNtZ2dxMkQxbS9zNDAKa2ovQTU3TWwyYXc3TTlEK1pqa0ZOQlZ2NEpvVVROQjlqREh5OFJLTDZCK2h1UXYxRnpNTnhIZGMrTFlIRmFydgp0ZDZGL3YvYmtXL2lScWpBeURNQ0F3RUFBYU1qTUNFd0RnWURWUjBQQVFIL0JBUURBZ0trTUE4R0ExVWRFd0VCCi93UUZNQU1CQWY4d0RRWUpLb1pJaHZjTkFRRUxCUUFEZ2dFQkFCRnVocUNVUG1UTmRJd282Q2QvL1U3blluQU8KTStKb0c3NVBrU1ZZWi9sUVFDY2xWVmVkdDgybmhGeEpoV0c0Z3dJOS9mZ2hmam9IMWI2UWdXWS80TEVVS0JTegpPNXZGaDROaEJvU1hsY0ozcXJxZ2ZoNFVHR1VNV21GRkl1d01lTjFRZzl5bUVMQnF1VS9FaXdRQjRDYWRwZDlPCldydWlySmNKNDEzcDRBUDZScGllVEVwb0thWGxZNXdiVElzblM3SC9ZR05nVm9Xa2pZSjBCdm1uY2hOYldndDMKS0VZbmxtQ25rRVorM1Y0WkgzYXp4M2psT1NreFFlbmwrdE5scmxmQWxyTDNCc25JWURiNmEybDRVd2hydXVwRQpJd1NzVXE2NDZ4WGpVRG1WMHA5NjhWakVVNzd2VWZrNUxESlpLb3BtS2RDZGhoc0F1c3JGejIyWFRDdz0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=
            key: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFcFFJQkFBS0NBUUVBMDV4V0Q4RDNMZzNrSk4wYVFuVlFjeldSUTF4WW9PNWtEMkwwWEF4T2FJcUo4S1ljCk5taWpjOGJkTVFQQXNGQTBHY2xGemtuR003emtNT21mMFYxYzhtbGdrQWNvc0w0cXR4ck5RTmhhQ09MVUpZTUEKdGR5eVczeE11NXlyK2gyT0ZBendJZzNnMFNtbHN6OTczS2g0ekRjdGxaQk00ekp6SXRYSEFXalprNm9XWllFOQpDR0prbmIwSGMvdjRrMkhybkZtVUVEaVZpaXVhLzBRZlBIdlFVYWRKZ1hKTFJGdnp1dFRQdlkzdmNVSUs5MXBICityL0FESGVhQ0NyWVBXYit6alNTUDhEbnN5WFpyRHN6MFA1bU9RVTBGVy9nbWhSTTBIMk1NZkx4RW92b0g2RzUKQy9VWE13M0VkMXo0dGdjVnF1KzEzb1grLzl1UmIrSkdxTURJTXdJREFRQUJBb0lCQURSS3BzMi96cFUzNDQvawpmMis2MDhXVWtEQUlLdktoMW1JaS91V2NPT2dHakMzR3JxUVhXWVRydUk4N01TdWd0aTlGR0pYd2p5VUw0WXZnCnY1aWFMTFRPcTRrTDY5YzVOdzhHZFlBM3RwQUpsWWtyaFVwcm5qdVRUTmJ6MFYrK1cvVENlYmpBbXpTMHlQaXgKa0djbnpxb1FYSmhnRDAvNWtKQWtLY2hFWTdma1ZiS1BUVkhFekJYK205QzNPTWlla0M1K0ZjTWFBOFFNTVJreQpRQUw2alBISS9TR1ZXYnRQRlk4L0VLYnBabjlzdGdMdFpyekNJbElMVXluMC91WC83OFVuZ3gyVVhFSllqK0lqCmJoOFBiMlBzQ256YzBGSm9QWHl3V01tLzFWL3FNTDNrVk1hUTFwM24zeDNaOGprTERVQ1ZtY1IzSEFrSVkxVi8KelAycjJya0NnWUVBL1Izb1d0OExxMEN4MlRhb25RSFZjNlFoZm82Z0ZtdGx0ZGwyaTNyay9nY0I5b3pWMXF6cApCWlR6SGZjNVlOMVdSWjlWbTBsV2M0S1h5cDdtRGhKdS9IZEZKd2hPU2pRS3A0MXEyYWFvS3lOSFVEZTd2UnJMCmtobnVXdE10REp1OHA4L1JJeHNFdnVrdVM5ZWpuSjJsRVQwbFlKaGVvQU1YdVVVdHhFOGxNZzBDZ1lFQTFnVmwKWXBBK0pWelB0MlVRa0ZIM1VFclM5ZHRpakZXTDBRem1wUE56MDBBaWpSb0pEQVA4TUFNUVlBa0pWcy93TEpwbgpFSmZLSmp3TFkwVElaMnRlMnZwVTBaVmlIV29Kc1lRNmx0dWJnZDFWbGpTLy9jTDJWOWJiUEFYWXk4cjFmcDkvCm5DT25xT2o3WXZqa2J3QkJxN2xKNElVWndNVDFXT05KRHhOTWt6OENnWUVBdHJDTmNua21iUGFHNXlaaVVPQnYKOWNWelAyc2w5TWlUWXN1UW1sK2JSQlkrdm5zc0pJUXN0QkNyNE9iOWpRSjBNRkF1YzZSZE40WDhsUXhYTTdUdQpVbDZybE42VDAwNzRtYktpZW5HbFUyMWxIV3I4b0NMazU1Qzd6dVk0ejY3Z1hhYkxaakVzSGJjajZTMjlNMTg5Ck10SVZWa0RqbTA1Z0l5TGhRNTEwVlVrQ2dZRUFzbllkYkdySzUyelU2Q0FtQjdIUmYrcGtydzRZeHR3dWtrc24KcURRNVNOWVorWDdVUEdpMlNYTEVuTS9zTWErQ25pN0I4bHdmL0hIbExRbVY4bWJkMmNzVUh3OXBtUTFxdDlPQwo1M2lIMjJvc2krdkFqR0dkK1BENExyelJZbDREQjJzSWhiSlZnOHVDazZ6bkRvZ3dPbmx1MlFFajBGSnNJNHFpCnlTZFdteEVDZ1lFQTgrNHBpMndMSkU5Q1BUQ0haZVRDQ1pDdnFwTTd1enBOaVRuMDRaVDFPd2Q2MjMvR1c2Q3cKWmo3MGRKMjAyblU3WldZcTFWR0pITVVKMlhNaDVZeSs2VmJsV2ZnUEQxVjNhMkxmZHorSHE2bDh6cGlHTlNpSgpTeVhrWVVac3VTcHVRVGREbVN0cHdDcnVxd0FoK1k4VkV2NHZaNzB0QXNmV3UycEZDZHFBSUZjPQotLS0tLUVORCBSU0EgUFJJVkFURSBLRVktLS0tLQo=
          frontProxyCAKeyPair:
            cert: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUN1RENDQWFDZ0F3SUJBZ0lCQURBTkJna3Foa2lHOXcwQkFRc0ZBREFOTVFzd0NRWURWUVFERXdKallUQWUKRncweE9UQTFNVGN3TXpJNE16bGFGdzB5T1RBMU1UUXdNekk0TXpsYU1BMHhDekFKQmdOVkJBTVRBbU5oTUlJQgpJakFOQmdrcWhraUc5dzBCQVFFRkFBT0NBUThBTUlJQkNnS0NBUUVBbk00ejBJeGNHWHpOY1FVdUlyWnVGZisyClM3NUNxUWsvRkt0VlV0eEdWZ25kakVJaEY5alcxbGRNZllpWDZhKy8xaVVCRXNzYzdjQ2NneGVBM0dqVmMwUmsKY3g4UjVyRnhqTjNoNGxaOVdyM3FPYzFDKzhwcGRnMEtGZmEwOFZ6VUVOK002eUtVcUNXS1VIYWlOUUltckwxcwpwTmF4YkhrdGlDUjRoZ1FJeUZKaExPRGY4M3o4dTlLbkl2WTk3aDJXemxNT1lUQnVwSVlXeDlzMDBwNUdPaDQyCndKSWY2OVhIbnd6eFZoY0piR1I2bUJjenBtVHVZTXEwR080RlZzbXkySXNzaFpJTHZsanc3ejlHWnhFTGM3cGkKYXNYR25xQUdybWxZN1pTSi9FVXFWQVh4SGZyMzA2R0lZOU9MaFg3dkpGSldZRUE3Z0FveEd6bUxpS0N6SHdJRApBUUFCb3lNd0lUQU9CZ05WSFE4QkFmOEVCQU1DQXFRd0R3WURWUjBUQVFIL0JBVXdBd0VCL3pBTkJna3Foa2lHCjl3MEJBUXNGQUFPQ0FRRUFtQ0MrUlFsMkZKL3dRUkUwdFBneS9qQXE2eWpDYnUxTTFoRUxwa2srd3NlTlFMYmkKNVpIZnZGUCtSNHdUU0phZ1JRTU5wY1NYbjMwd1VvVGZ2VU5SRHZ4bDVKblhFQWNjVTZTY3cvQ0J3ZU9OWnlXTQo2eStiZHBQRXNsSmtRRWtmN3NjZWFUQThaRjh2TWhaZmFWNHV1NFhwRHVIdVRZRi9TSkY2V0toR3d2NERRYWJ1Cm1temtFRVlNOFkxWUNsM1hvSXNuOVo0UTRoTFl3SnZ0RFFzeHppMzJNVzAvTFN1R09xczdoZFlzaHNVRU5OdlkKQThmbEpZU3RBd2g0TnY5cDJCelBzV3Q3SlpnemFuQlNSblRhVEZESGtPZ1cyajNKL052ZFR5YjhOR0NRTzRZcQp0NS9MSlF5RGcrdnJlZGxIM2Y4b1M5R2k3SVl4dklJNkhuTjRwUT09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
            key: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFcEFJQkFBS0NBUUVBbk00ejBJeGNHWHpOY1FVdUlyWnVGZisyUzc1Q3FRay9GS3RWVXR4R1ZnbmRqRUloCkY5alcxbGRNZllpWDZhKy8xaVVCRXNzYzdjQ2NneGVBM0dqVmMwUmtjeDhSNXJGeGpOM2g0bFo5V3IzcU9jMUMKKzhwcGRnMEtGZmEwOFZ6VUVOK002eUtVcUNXS1VIYWlOUUltckwxc3BOYXhiSGt0aUNSNGhnUUl5RkpoTE9EZgo4M3o4dTlLbkl2WTk3aDJXemxNT1lUQnVwSVlXeDlzMDBwNUdPaDQyd0pJZjY5WEhud3p4VmhjSmJHUjZtQmN6CnBtVHVZTXEwR080RlZzbXkySXNzaFpJTHZsanc3ejlHWnhFTGM3cGlhc1hHbnFBR3JtbFk3WlNKL0VVcVZBWHgKSGZyMzA2R0lZOU9MaFg3dkpGSldZRUE3Z0FveEd6bUxpS0N6SHdJREFRQUJBb0lCQUFOUmplRXRCMG4yelRaRwpJTXJWUjVFcG4wY05HTVlSRHdlMTlKRlRYaDIyQ2IxTkxQd2ZON1REbGpmVjZ6a2o0aEI3S2dHbTBNN3JVNlNtCm03Q09lMjM4RlpBbUtTL1RzNDZDcDZRdHBtdUVOMi9QdTBvdTUzcDdIaXFHMVIrQ2ttNWsvTXVCS05wQ0tTSTQKMElnRXFxTGZRMnhkcXRXYjN1M1JyOGRPVUkxRXdKUzBDTitIYXp5ajVrZlF6cmpPTDV1eEFLbXVTU1lFOCsvawpBL082Vm9sKzZMeVYrSUxQelhBd1RPSVJIcWhPby8xQUtldnpyb3hTYXRlb0p0dUxhaW05Ri9RclZCVDFneFluCjdQcy8yeEwxRHhBNTNEMjVqa1ZtYk83WGRzbEw2NUdPZy9ITlkrbEE1ZDRPTXNEWGUrdVRqN1BJU3E3UWhHWDIKRm4wam5Za0NnWUVBeUlvdGMyR0JFdVFvQW9TaElLN3pEMmQybE9mdFRmd2tMeDRkQ3NOMHRkSTlVaTZySWowMQpENmhyNHpncTRDZmlmaTJnUm5KZDFzVXNIVUNwRGgwdUZPMDY1MFFBMXA5cTNaSVJWUWYvSVBkQzVIWWR3UVh0Ck1WTGlJaHpjMFAwczdmeEVSSXgvbENXVHdIRG5DK2ZPeTI0UHArL1NieXMwbmwwWUhoQk4zK3NDZ1lFQXlDdTUKT1ZjTlRtdzRsOEJHL05Wc2gyeW5ueTlDUEhpK2tWQU0waEQwVzlWdWFub2xpdXRMTjFHcnZ6eSt6cisvQmpmZApKYWtlY01GSFFiNXVOdlZFTHlYMXMvTkJ5NjErWnRueWZGSXVHSHFZdGdOQVpDeGtrM2xNMUFVL1JEVUxKL08wCkc1OFVGUjVPNll5b3k5WWZldEY4OUZRQmpiNEF1VHJQbEY5aklKMENnWUVBam51K3QwLzd5VlJhS1EvYSs4SFIKNkl2MmNPNG9hVlJRMFRsd0lRbW1qdGtGd0xKdjNTL24xMnd1MjQ0NHlITU9OZUJ0RkNDR0UrYWI1Vnpmd0t0eQo1bU4zaW9HQ3B2czFqcUFOdUlDcUFONHRwTzFYVHFITFdWUXVYMVpxZmdLa1BhTVRUakVWSkVsZXBVaVNvSjdmCkN5THo5TG9zcGRmbzF1d0dDclpDM21rQ2dZRUF2cDk4M2RFNzE4SVJ4dG9TQURjVENvaDd2SWxaejVMQkVFc20KV21wUStwOXZiakR5VGJBelNmUVoxWjE0ckJWSVNoaXJIbkZHanVSUkFwZmlCNjVjaDNYajNjRzdsOGFaeUVLbgp2S0xhU085L1BGNHVWUGM5dEg5Z25jeDlhbXdGT3IvSGRrSnc4b2VSYUxKT0VRZlJwTG1aQUdoN3Jrc1NEMU9sCldNdlo3N1VDZ1lBNkhjT3huUU5WSGJhN21rcHp3T0hOVDI3c0NEbndGaDdId3o4M0s3U01oS3dWSTV6NTdxTlMKQm5wNm9vOHd1NTRyTXExU3FCSEo1OWFPS0t6K3dGV1pxdzhwaVFkQ2l5WGt0VGRxQVlmdkJtMmFoTVZiVHVUQwo4RWNJRmN6aU5zVFQyc3psdjZBS3M0WDFLdE8vZDIrY21BKzdTYUhoWmh6SUtaSkt4RUdDNlE9PQotLS0tLUVORCBSU0EgUFJJVkFURSBLRVktLS0tLQo=
          saKeyPair:
            cert: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUM1RENDQWN5Z0F3SUJBZ0lCQURBTkJna3Foa2lHOXcwQkFRc0ZBREFqTVNFd0h3WURWUVFERXhoellTMWoKWlhKMGFXWnBZMkYwWlMxaGRYUm9iM0pwZEhrd0hoY05NVGt3TlRFM01ETXlPRE01V2hjTk1qa3dOVEUwTURNeQpPRE01V2pBak1TRXdId1lEVlFRREV4aHpZUzFqWlhKMGFXWnBZMkYwWlMxaGRYUm9iM0pwZEhrd2dnRWlNQTBHCkNTcUdTSWIzRFFFQkFRVUFBNElCRHdBd2dnRUtBb0lCQVFDcEtXc1FwblBjMWU0K01na0E1blY4cTFaZm9EMjMKQ1FpTE1BaTMrVDBxVmhCQlU2MTFiVUR2dXZCME4zUGc2TktmL1NqdkJzZEY2dnkzdDg0YWxiOXVmMmVvaTg1ZApzc2RRaWp5WVRyWjBBMFhOUkdVd1Z5c3JpSmhaVUFhRFdGTnZWOTlRc3FLUGtvNTl4M1NCbnN0SHhsaW5mVUNSCnpLVElydHJReDB0bTRNdnBQbS9QZjQ3c3dvbjQvaDJpdEpQa3FubVJ0Z1Z1WkVwR3RMWUZGdkNDWTRNTnp3bHAKSzN6K1FMRzVyMnNqNHRMVTA4bDYyL1NxWnpUOTB0Mmk3N2NsaEI4dkVYZlJMN1dPdG4rR29LQThEUHZIY0c5UgoyTjJsS1NZcnZPWm5iMG9FYldrdStsdVN6STBJaGppNUU1U2xGMGxqM0ZLcjV0Rmd1dThIVTg0YkFnTUJBQUdqCkl6QWhNQTRHQTFVZER3RUIvd1FFQXdJQ3BEQVBCZ05WSFJNQkFmOEVCVEFEQVFIL01BMEdDU3FHU0liM0RRRUIKQ3dVQUE0SUJBUUFTZjdZQ0dsdjZiRmluVEdUQXp2bmZmejZmNnpSa0prUFNHK1pHTU90MHhnQllsaElkS0huQgpQOFdZT1VsUzB3MGx2ZDMvMElTL1JIU1N4VDZpNFVvOFdNSWlVNDJkWkE4SzBrNHR1K3R2TjFuMjlGbjVRa2hVCjRZYkx1RCs2L0dDMmhtTzQzbjhTMUx0aTNPRm1KUEtOc2dGZGx0VlBQMy93akpCR3JWNVZBVDRBRnJwZkN6WXMKSk0yWFZnUHd0dUlJKzczTUxRUjNIU3pXRXJ2SnF0OE5ONk5xR1IwVE1XTFJYNXdtTkVFTVNWZXpLY0o3ZEhHaQpGSEVwMGM2dnhHbWVROFltQUw3Ulc3ZjY5RGlwUXdFNnVmL1RlZ3BiRmlDUEhCRmdjejNjZXB5SmVZcG9yaktuCmMwZWQ4Vi9kbEU5U2dxMEs5VVV1T2ZUVDNNQjNwcVBqCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
            key: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb3dJQkFBS0NBUUVBcVNsckVLWnozTlh1UGpJSkFPWjFmS3RXWDZBOXR3a0lpekFJdC9rOUtsWVFRVk90CmRXMUE3N3J3ZERkejRPalNuLzBvN3diSFJlcjh0N2ZPR3BXL2JuOW5xSXZPWGJMSFVJbzhtRTYyZEFORnpVUmwKTUZjcks0aVlXVkFHZzFoVGIxZmZVTEtpajVLT2ZjZDBnWjdMUjhaWXAzMUFrY3lreUs3YTBNZExadURMNlQ1dgp6MytPN01LSitQNGRvclNUNUtwNWtiWUZibVJLUnJTMkJSYndnbU9ERGM4SmFTdDgva0N4dWE5ckkrTFMxTlBKCmV0djBxbWMwL2RMZG91KzNKWVFmTHhGMzBTKzFqclovaHFDZ1BBejd4M0J2VWRqZHBTa21LN3ptWjI5S0JHMXAKTHZwYmtzeU5DSVk0dVJPVXBSZEpZOXhTcStiUllMcnZCMVBPR3dJREFRQUJBb0lCQUI3RnpCWkJVWDV3NUdBbwpGZjgxN1ZWNmpjSGprcGFEYkN4MTFvQXhOUEZJcXJoWGtveDBEWVlPeWNNNmV6Z0U0cHY4SDhBcnlZQnNtUUNLCnpWR0V3RWhIb1FIR1BRcEtoWHVmU2hxaTV3bi90bWo2OGpWekJnVnJXZHVWZFRuYmpZSUp5RFFUNndLWE5KaW8KK2diQ2JsUm1QcVpwWUorbFRLeTlNazBjbEJqb3JEamFMcHdUSDRQbllyQkZTZzJyWHlIbzB6UnlVaEtTbVhKNgpkaGx0ZmVyNTl1QWxRQnV4QVFuVWRucjNQTVpZMkhtem55UEwxZ1dzZGxaSTBXcCtQam1TeS8zdmNTQlFHczhtClplZGNweEYxN2tka3Q4L3V5OGRSUVVLRCtGcUU3SkNCSUllL3UwSDFNVFc1NTV1ZEx5ZmY3N21CNEE4OUVqcmkKYi9DbExvRUNnWUVBMFBEMDNEZW8zL3U4SXY3SEdzV0Y1bzlaSGNCcDZieHBaeGFRZHhuQXhzdjFyTjh5a3pDOQpnSHNLYjR5UUJTQTNhNUwzYU5OVDV0d1loUXBzTVorenh6WjgyaThQRmwrZGYvazUyenc4VmhFejdMYUVQblVCCk16aTB0aWFkUXdJNUVmbFhKaWVrMU5YWHZyaDhWRDVSSmxxdUkyblVnWmkxV3lhYzVEck9EeUVDZ1lFQXowTGcKSi9ja0g1djVFNHFEWjZNaHhxL09PSjRkdHk3UElBL2tLSUVDOHJCZ1Q5aGZqeWJYUmhxUVR5aDlsWkIwUGtaMApOMFJrUGJOMHNiSU9RTCt3REJ3dmVlVEFPS1ovTkFMTkRvQXczOVREVlhnRG5RZDVmZUxyam84MS9vOE9lSVJxClh0YTJ1K2xsUVlROG9EUzlwYllDYmprMllSZGdxN0lNK1pqcm9ic0NnWUJjM1R0M1JTWEJwMWtQRkwzWm9FREwKSUpzekpmbnM4TmpJQUxka3VBVitWZGh6WlNCTld6UmVqbEV0RXdSUHd1bmUzZ3NvaEFTZWJ1UlcvVExwTzFuawpDTXVsRFpWZkZGQWtPTmtHSDllUlNVUVN5V3d0ZGtONlNKSEpBNUNSMzhNTndneUI0TXpaNjlGZjZ3OFhRanMvCkdMNmM3c1NNZFJybDBGdWE5S2Z4QVFLQmdCTE1SZmhaK2ZURCtMdEUvTllSZmFhL216eVhXcXFhbkQ2VU1tVmEKRGlKa3pOZHhFSG16VkNNUGxiY1lQUXVycGw5ZmxIck93U2kzZGdZSDJETVhMNmhwaGdUUU1uN3cydWlrdUdSdwpTLzZCRlpaUzVFRUJ4SXNlWWE3MFhqbFFVRWV0K3RmUE1aT3BmMzJKdU5YdThxUnM5WnQ1cE96NWFkTW91dlNJClloYXhBb0dCQUtRT1BtMW1RL0tZdHBCWjE2djdKak5ieW5iNmNETjFHZ1VOYVZ3bjdDbTZTbVRVYXMxbitoMC8KdElMYlpmUWdxSHN1LzFaQk0vY2xNZU9NVGV6N2J3TmJhY01HTVphU2JTQkJsay9CRmRuV3NOSndydy9UOHFObAphbW9UaitVam1GSTEzS1BCWjY3aHVkb01ieWdHNGkvSTIyc2J2NFlsSlpKdGp4U1JucXQwCi0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg==
          clusterConfiguration:
            etcd: {}
            networking:
              serviceSubnet: ''
              podSubnet: ''
              dnsDomain: ''
            kubernetesVersion: ''
            controlPlaneEndpoint: ''
            apiServer: {}
            controllerManager: {}
            scheduler: {}
            dns:
              type: ''
            certificatesDir: ''
            imageRepository: ''
    status:
      apiEndpoints:
      - host: 35.222.174.181
        port: 6443
      providerStatus:
        apiVersion: gceproviderconfig/v1alpha1
        kind: GCEClusterProviderStatus
        metadata:
          creationTimestamp: 
  config:
    masterCount: 3
    cloud:
      cloudProvider: gce
      project: ackube
      region: us-central1
      zone: us-central1-f
      instanceImage: ubuntu-1604-xenial-v20170721
      os: ubuntu-1604-lts
      instanceImageProject: ubuntu-os-cloud
      networkProvider: calico
      ccmCredentialName: gce
      sshKeyName: g1-sshkey
      gce:
        NetworkName: default
        NodeTags:
        - g1-node
    kubernetesVersion: v1.13.5
    caCertName: ca
    frontProxyCACertName: front-proxy-ca
    credentialName: gce
    apiServerExtraArgs:
      cloud-config: "/etc/kubernetes/ccm/cloud-config"
      cloud-provider: gce
      kubelet-preferred-address-types: ExternalDNS,ExternalIP,Hostname,InternalDNS,InternalIP
    controllerManagerExtraArgs:
      cloud-config: "/etc/kubernetes/ccm/cloud-config"
      cloud-provider: gce
status:
  phase: Ready
  cloud:
    loadBalancer:
      dns: ''
      ip: 35.222.174.181
      port: 6443
```
Here,

  - `status.phase`: is ready. So, you can use your cluster from local machine.
  - `status.apiserver` is the cluster's apiserver address


To get the `kubectl` configuration file(kubeconfig) on your local filesystem run the following command.
```console
$ pharmer use cluster g1
```
If you don't have `kubectl` installed click [here](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

Now you can run `kubectl get nodes` and verify that your kubernetes 1.13.5 is running.
```console
$ kubectl get nodes
kubectl get nodes
NAME                       STATUS   ROLES    AGE     VERSION
g1-master-0                Ready    master   6m21s   v1.13.5
g1-master-1                Ready    <none>   3m10s   v1.13.5
g1-master-2                Ready    master   2m7s    v1.13.5
n1-standard-2-pool-5pft6   Ready    node     56s     v1.13.5

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
NAME                       AGE
g1-master-0                3m
g1-master-1                3m
g1-master-2                3m
n1-standard-2-pool-5pft6   3m


$ kubectl get machinesets
NAME                 AGE
n1-standard-2-pool   3m
```
#### Create new master machines

You can create new master machine by the deploying the following yaml

```yaml
kind: Machine
apiVersion: cluster.k8s.io/v1alpha1
metadata:
  name: g1-master-3
  creationTimestamp: '2019-05-17T03:28:40Z'
  labels:
    cluster.k8s.io/cluster-name: g1
    node-role.kubernetes.io/master: ''
    set: controlplane
spec:
  metadata:
    creationTimestamp: 
  providerSpec:
    value:
      kind: GCEMachineProviderSpec
      apiVersion: gceproviderconfig/v1alpha1
      metadata:
        creationTimestamp: 
      roles:
      - Master
      zone: us-central1-f
      machineType: n1-standard-2
      os: ubuntu-1604-xenial-v20170721
      disks:
      - initializeParams:
          diskSizeGb: 30
          diskType: pd-standard
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
  creationTimestamp: '2019-05-17T03:28:40Z'
  labels:
    cluster.k8s.io/cluster-name: g1
    node-role.kubernetes.io/master: ''
    set: node
spec:
  providerSpec:
    value:
      kind: GCEMachineProviderSpec
      apiVersion: gceproviderconfig/v1alpha1
      roles:
      - Master
      zone: us-central1-f
      machineType: n1-standard-2
      os: ubuntu-1604-xenial-v20170721
      disks:
      - initializeParams:
          diskSizeGb: 30
          diskType: pd-standard
  versions:
    kubelet: v1.13.5
```
#### Create new machinesets

You can deploy new machinesets by deploying the following yaml

```yaml
kind: MachineSet
apiVersion: cluster.k8s.io/v1alpha1
metadata:
  name: n1-standard-2-pool
spec:
  replicas: 1
  selector:
    matchLabels:
      cluster.k8s.io/cluster-name: g1
      cluster.pharmer.io/mg: n1-standard-2
  template:
    metadata:
      labels:
        cluster.k8s.io/cluster-name: g1
        cluster.pharmer.io/cluster: g1
        cluster.pharmer.io/mg: n1-standard-2
        node-role.kubernetes.io/node: ''
        set: node
    spec:
      providerSpec:
        value:
          kind: GCEMachineProviderSpec
          apiVersion: gceproviderconfig/v1alpha1
          metadata:
            creationTimestamp: 
          roles:
          - Node
          zone: us-central1-f
          machineType: n1-standard-2
          os: ubuntu-1604-xenial-v20170721
          disks:
          - initializeParams:
              diskSizeGb: 30
              diskType: pd-standard
      versions:
        kubelet: v1.13.5
```

#### Create new machine-deployments

You can deploy new machine-deployments by deploying the following yaml

```yaml
kind: MachineDeployment
apiVersion: cluster.k8s.io/v1alpha1
metadata:
  name: n1-standard-2-pool
spec:
  replicas: 1
  selector:
    matchLabels:
      cluster.k8s.io/cluster-name: g1
      cluster.pharmer.io/mg: n1-standard-2
  template:
    metadata:
      labels:
        cluster.k8s.io/cluster-name: g1
        cluster.pharmer.io/cluster: g1
        cluster.pharmer.io/mg: n1-standard-2
        node-role.kubernetes.io/node: ''
        set: node
    spec:
      providerSpec:
        value:
          kind: GCEMachineProviderSpec
          apiVersion: gceproviderconfig/v1alpha1
          metadata:
            creationTimestamp: 
          roles:
          - Node
          zone: us-central1-f
          machineType: n1-standard-2
          os: ubuntu-1604-xenial-v20170721
          disks:
          - initializeParams:
              diskSizeGb: 30
              diskType: pd-standard
      versions:
        kubelet: v1.13.5
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
$ pharmer delete cluster g1
```
Then, the yaml file looks like

```yaml
$ pharmer get cluster g1 -o yaml
kind: Cluster
apiVersion: cluster.pharmer.io/v1beta1
metadata:
  name: g1
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

Now, to apply delete on provider cluster run
```console
$ pharmer apply g1
```

**Congratulations !!!** , you're an official `pharmer` user now.
