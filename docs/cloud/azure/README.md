---
title: Azure Overview
menu:
product_pharmer_0.3.1
identifier: azure-overview
name: Overview
parent: azure
weight: 10
product_name: pharmer
menu_name: product_pharmer_0.3.1
section_menu_id: cloud
url: /products/pharmer/0.3.1/cloud/azure/
aliases:
- /products/pharmer/0.3.1/cloud/azure/README/
---

# Running Kubernetes on [Azure](https://azure.microsoft.com)

Following example will use `pharmer` to create a Kubernetes cluster with 1 worker nodes and 3 master nodes (i,e, 4 nodes in you cluster).

### Before you start

As a prerequisite, you need to have `pharmer` installed.  To install `pharmer` run the following command.

```console
$ mkdir -p $(go env GOPATH)/src/github.com/pharmer
$ cd $(go env GOPATH)/src/github.com/pharmer
$ git clone https://github.com/pharmer/pharmer
$ cd pharmer
$ ./hack/make.py

$ pharmer -h
```

### Pharmer storage

To store your cluster  and credential resource, `pharmer` use [vfs](/docs/cli/vfs.md) as default storage provider. There is another provider [postgres database](/docs/cli/xorm.md) available for storing resources.

To know more click [here](/docs/cli/datastore.md)

In this document we will use local file system ([vfs](/docs/cli/vfs.md)) as a storage provider.

### Credential importing



**Tenant ID:**
From the Portal, if you click on the Help icon in the upper right and then choose `Show Diagnostics` you can find the tenant id in the diagnostic JSON.

You can also find TenantID from the endpoints URL

![azure-api-key](/docs/images/azure/azure-api-key.png)

From command line, run the following command and paste the api key.
```console
$ pharmer create credential azure --issue
```
![azure-credential](/docs/images/azure/azure-credential.png)

Here, `azure` is the credential name, which must be unique within your storage. With `issue` flag you can issue new credential.
If you want to use your existing credential then no need to pass `issue` flag.

To view credential file you can run:

```yaml
$ pharmer get credential azure -o yaml
apiVersion: v1alpha1
kind: Credential
metadata:
  creationTimestamp: null
  name: azure
spec:
  data:
    clientID: <client id>
    clientSecret: <client secret>
    subscriptionID: <subscription id>
    tenantID: <tenant id>
  provider: azure
```

Here,
 - `spec.data.clientID` is the azure client id
 - `spec.data.clientSecret` is the secret
 - `spec.data.subscriptionID`  is the subscription id of azure account
 - `spec.data.tenantID` is tenant id that you provided which can be edited by following command:
 
```console
$ phrmer edit credential 
```

To see the all credentials you need to run following command.

```console
$ pharmer get credentials
NAME         Provider       Data
azure        Azure          tenantID=77226, subscriptionID=1bfc, clientID=bfd2fee, clientSecret=*****
```
You can also see the stored credential from the following location:
```console
~/.pharmer/store.d/$USER/credentials/azur.json
```

You can find other credential operations [here](/docs/credential.md)


### Cluster provisioning

There are two steps to create a Kubernetes cluster using `pharmer`. In first step `pharmer` create basic configuration file with user choice. Then in second step `pharmer` applies those information to create cluster on specific provider.

Here, we discuss how to use `pharmer` to create a Kubernetes cluster on `azure`

#### Cluster Creating

We want to create a cluster with following information:

- Provider: azure
- Cluster name: azure
- Location: eastus2
- Number of master nodes: 3
- Number of worker nodes: 1
- Worker Node sku: Standard_B2ms (cpu: 2, memory: 4 Gb)
- Kubernetes version: v1.13.5
- Credential name: [azure](#credential-importing)

For location code and sku details click [hrere](https://github.com/pharmer/cloud/blob/master/data/json/apis/cloud.pharmer.io/v1/cloudproviders/azure.json)

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
$ pharmer create cluster azure-1 \
    --masters 3 \
    --provider azure \
    --zone eastus2 \
    --nodes Standard_B2ms=1 \
    --credential-uid azure \
    --kubernetes-version v1.13.5
```

To know about [pod networks](https://kubernetes.io/docs/concepts/cluster-administration/networking/) supports in `pharmer` click [here](/docs/networking.md)

The directory structure of the storage provider will be look like:


```console
$ tree ~/.pharmer/store.d/$USER/clusters/
/home/<user>/.pharmer/store.d/<user>/clusters/
├── az1
│   ├── machine
│   │   ├── az1-master-0.json
│   │   ├── az1-master-1.json
│   │   └── az1-master-2.json
│   ├── machineset
│   │   └── standard-b2ms-pool.json
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
│       ├── id_az1-sshkey
│       └── id_az1-sshkey.pub
└── az1.json

6 directories, 15 files
```


Here,
  - `machine`: conntains information about the master machines to be deployed
  - `machineset`: contains information about the machinesets to be deployed
  - `pki`: contains the cluster certificate information containing `ca`, `front-proxy-ca`, `etcd/ca` and service account keys `sa`
  - `ssh`: has the ssh credentials on cluster's nodes. With this key you can `ssh` into any node on a cluster
  - `az1.json`: contains the cluster resource information

You can view your cluster configuration file by following command.

 
```yaml
$ pharmer get cluster az1 -o yaml
apiVersion: cluster.pharmer.io/v1beta1
kind: Cluster
metadata:
  creationTimestamp: "2019-05-16T08:56:19Z"
  generation: 1557996979574735025
  name: az1
  uid: 78caadfe-77b8-11e9-991c-e0d55ee85d14
spec:
  clusterApi:
    apiVersion: cluster.k8s.io/v1alpha1
    kind: Cluster
    metadata:
      creationTimestamp: null
      name: az1
      namespace: default
    spec:
      clusterNetwork:
        pods:
          cidrBlocks:
          - 192.168.0.0/16
        serviceDomain: cluster.local
        services:
          cidrBlocks:
          - 10.96.0.0/12
      providerSpec:
        value:
          apiVersion: azureprovider/v1alpha1
          caKeyPair:
            cert: null
            key: null
          clusterConfiguration:
            apiServer: {}
            certificatesDir: ""
            controlPlaneEndpoint: ""
            controllerManager: {}
            dns:
              type: ""
            etcd: {}
            imageRepository: ""
            kubernetesVersion: ""
            networking:
              dnsDomain: ""
              podSubnet: ""
              serviceSubnet: ""
            scheduler: {}
          etcdCAKeyPair:
            cert: null
            key: null
          frontProxyCAKeyPair:
            cert: null
            key: null
          kind: AzureClusterProviderSpec
          location: eastus2
          metadata:
            creationTimestamp: null
            name: az1
          networkSpec:
            vnet:
              name: ""
          resourceGroup: az1
          saKeyPair:
            cert: null
            key: null
          sshPrivateKey: ""
          sshPublicKey: ""
    status: {}
  config:
    apiServerExtraArgs:
      cloud-config: /etc/kubernetes/azure.json
      cloud-provider: azure
      kubelet-preferred-address-types: ExternalDNS,ExternalIP,InternalDNS,InternalIP
    caCertName: ca
    cloud:
      azure:
        azureDNSZone: cloudapp.azure.com
        azureStorageAccountName: k8saz1by5v4x
        controlPlaneSubnetCIDR: 10.0.0.0/16
        internalLBIPAddress: 10.0.0.100
        nodeSubnetCIDR: 10.1.0.0/16
        resourceGroup: az1
        rootPassword: QpNqy8m14iqzEand
        subscriptionID: 1bfc9f66-316d-433e-b13d-c55589f642ca
        vpcCIDR: 10.0.0.0/8
      ccmCredentialName: azure
      cloudProvider: azure
      networkProvider: calico
      region: eastus2
      sshKeyName: az1-sshkey
      zone: eastus2
    credentialName: azure
    frontProxyCACertName: front-proxy-ca
    kubernetesVersion: v1.13.5
    masterCount: 3
status:
  cloud:
    loadBalancer:
      dns: ""
      ip: ""
      port: 0
  phase: Pending
```


You can modify this configuration by:
```console
$ pharmer edit cluster az1
```

#### Applying 

If everything looks ok, we can now apply the resources. This actually creates resources on `azure`.
Up to now we've only been working locally.

To apply run:

 ```console
$ pharmer apply az1
```

Now, `pharmer` will apply that configuration, this create a Kubernetes cluster. After completing task the configuration file of the cluster will be look like

 
```yaml
$ pharmer get cluster az1 -o yaml
kind: Cluster
apiVersion: cluster.pharmer.io/v1beta1
metadata:
  name: az1
  uid: 78caadfe-77b8-11e9-991c-e0d55ee85d14
  generation: 1557996979574735000
  creationTimestamp: '2019-05-16T08:56:19Z'
spec:
  clusterApi:
    kind: Cluster
    apiVersion: cluster.k8s.io/v1alpha1
    metadata:
      name: az1
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
          kind: AzureClusterProviderSpec
          apiVersion: azureprovider/v1alpha1
          metadata:
            name: az1
            creationTimestamp: 
          networkSpec:
            vnet:
              name: ''
          resourceGroup: az1
          location: eastus2
          sshPublicKey: c3NoLXJzYSBBQUFBQjNOemFDMXljMkVBQUFBREFRQUJBQUFCQVFDM0VudlByUDVBYkJWVnB4REVjUm9WSWJnL1VyV2xiSkNhVkNmei9EaittR0pVbjA3b0FLdXhOQ2JXTTF6a2JXTFh4V1ZGa3laeDZTNEhuK1VDK1JqL1k2VXFYclk5MS9weWhSZUkzNGl2WmNrUkZZdlRHcE9raXZ6ZFdMT0tjajZsWjh2SGF0MnVON2R3N284UWsvd09TNDJzRzBBUk83d0JPbG5GcEZ3VWkrWE41NXNoK210eENtREoxNTVqdGd1QWtnUENiRHpvYXdCZEs2bnRNSUU3bklpOVJ4ODlSVENwZk1lU3VsUnpVOGNvMmRQMVBtNndYWDRDYTkydzE1K3d3b2F1SFJKNTB2eGppdXNwTENhTngwU2hiWlRJdWFFUVZkMVdyTXEzaEQ0U3lBWTBlZUIzZXVueFE5Z2ZVS20zbHJlLzdtRTVyaGk2V3E0TDVyV3o=
          sshPrivateKey: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb3dJQkFBS0NBUUVBdHhKN3o2eitRR3dWVmFjUXhIRWFGU0c0UDFLMXBXeVFtbFFuOC93NC9waGlWSjlPCjZBQ3JzVFFtMWpOYzVHMWkxOFZsUlpNbWNla3VCNS9sQXZrWS8yT2xLbDYyUGRmNmNvVVhpTitJcjJYSkVSV0wKMHhxVHBJcjgzVml6aW5JK3BXZkx4MnJkcmplM2NPNlBFSlA4RGt1TnJCdEFFVHU4QVRwWnhhUmNGSXZsemVlYgpJZnByY1FwZ3lkZWVZN1lMZ0pJRHdtdzg2R3NBWFN1cDdUQ0JPNXlJdlVjZlBVVXdxWHpIa3JwVWMxUEhLTm5UCjlUNXVzRjErQW12ZHNOZWZzTUtHcmgwU2VkTDhZNHJyS1N3bWpjZEVvVzJVeUxtaEVGWGRWcXpLdDRRK0VzZ0cKTkhuZ2QzcnA4VVBZSDFDcHQ1YTN2KzVoT2E0WXVscXVDK2Exc3dJREFRQUJBb0lCQUJOSjNrT21UVytLTThFLwpoZk84bXV2cERwbVZaRkFXblRHMWRqUXR1ZStSTEtNUDJlZDEwcUVzQm4rQkQrTjladkdtK2FHWC9HLzZDb0NCCkowYmw2ZTFXbVZ0YWVVY1F6M0ZyZG14VWFQbFo5eEpXdTlHMU5pTWJCY05vaWhvbktWU1NHQlZkdkJlVUJUN2YKMDdFQ2RvY25ETGs2Y2NpZkM1THhpKzNZQUYrbHBCVk84enZ2WVZRUXllMHFWTTFEYlkwa09GZjBmK0xFbklxNgpUQXFTZi9waWlKcUpIM25lWUlqR21heVJCNGtPUS9OQmVuWnorUFVkKzU2RXJ0Z0dScTMwWW90eXBEejRxKzA5CkhtbzdXdEFIMmhVNWhLVGEzUUc0bkdBbEpVRmQ3ZFFxQzFTTzhrZWNMWENZdFUrWlY3NjBWb2ZnVDBlRE1GVFEKUmg0R0Jla0NnWUVBNUNLSXl1b1VyUXg3REZMaVBCTitKTm90bzgza2k3ZUxpclBsanY0bGwvd3pvZ0xYUVRpTwptZDF1dmk5U2d2U2RQcEVCWTZVRmxhamlLSS9oN011UkMyY3FCdjdqazZWazkvb2FsVjBWVXRCZk9LQ29wb2doCktWSDFCVTc2SU43RHlrQUpHQ0NhaGNFbldtekFzdXdlQWZJZkRFKzF0a2hHOVhJVFYweWgwNjBDZ1lFQXpXN20KaGZaVHFQdjMxKzdQcExFZmNFbHE0Y2l0TjdBRk1LY2hKb0dCRmFGWGc0T2RJb2hWWFZOb3ZkTDVWUkhMbDNwUgpkZW9IUTRxSzZqV0tPV01qcy9wS2NER0k2N3R4NEhQdDRlMFlGZ1BTMFVZbWpZeElkMGJ3SzYzcng5SXV4M1hFCmVReEREY3lFWXRCbUFoUTNZMTN6ditWRmtGMzZHM1dpc1hVbTJ0OENnWUJxYm80aEZLb0d2YzdlUmdEVUJFZ1MKaTFORm0zWG5sUDdkKytXNkcybVFpWkhSSU1BcDVtZm84cnlLcitzdnUwMXM5aHVPMEZ0Vm9nKzQycitOU0w5bgpjWDdTK3JGVG5aTUllYjlUTmJVUUNMU1Q1NmdtNFZXUFFIUXVRTlZDNW9xelhjS2daZjJSTHpiYjRlYlkwbjJCCmJPTDlUR2E3SHVjejlUOSt0L3E3bFFLQmdDaS8vdXBEMm9TQ3RyOFdtQW5MT0xsRlZ2WkNvRm1UaVBRRnN3VzQKV3FxM3ZteFFCek42WjdTRGZ4dG9aaDBCMHFqUmtxY1pMU2V3cTYyWndUbHcrUHdTZ2dHUFVlR3c1UDNwQVI3MwpzUGRzK3J5WWRiMU9QbkdxbUttUmJsdk16WXF6U2EzWlNOUEw1ZGJVRCtnSnFwTURaLzZBdERQVzhHM1IvOXZECnFWbHhBb0dCQU5WMFFFeTM0ZFNLbjdqSVV4NzdLM0k4S3hraFFmS2pzQXRiS3dsQmtkVFNJVjhER3RvZUlXcmUKK2hXWVA1RXk1RWJnYXhuRFpPblN2d05LSy80VGU4TmJPdHJuRVBRUjhGMUdrWVkyZjAyU1pmTDNiY1RZOThFeQo5ZERpbG1RbmFEOWp4R2hFTVducUNUUVRVeWZRVFlUdk1TTjNxRExPYkNLaHJoclR1a0N1Ci0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg==
          caKeyPair:
            cert: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUN1RENDQWFDZ0F3SUJBZ0lCQURBTkJna3Foa2lHOXcwQkFRc0ZBREFOTVFzd0NRWURWUVFERXdKallUQWUKRncweE9UQTFNVFl3T0RVMk1qQmFGdzB5T1RBMU1UTXdPRFUyTWpCYU1BMHhDekFKQmdOVkJBTVRBbU5oTUlJQgpJakFOQmdrcWhraUc5dzBCQVFFRkFBT0NBUThBTUlJQkNnS0NBUUVBcXVyREJkM3piSXduMU12bFZ2Y2Qzc3JJCmFobnh3RThHd3M2ajNOUldQd1dzeFlFUStKN0pGdEtlS0NEN0tLdTlYMEVRS0V1QWdFbjZYSm5lZkVReGJHNjYKNGxKZ0NXSDQzYjNURWhBZU9NNDdKZURFak9wZFUwZk9nb0o5SkdSZmpCcHF1dW9MVkUrRmxMU2dmanY1S0k0TAo1OFU4NHpwUWdDK3dTSDVhOVdDVy9Pc0RXZkZMeFRzSUR5aW40aGFQNDdCMXQvUnlxQ2NkYXROTldVWDFCSEdYCkZ6alhVUURic3pKRjBxS3lhay9Ub0JKalJ5NGNFb0s5TnhSYXR3RFZxcUxtanBFY0pBQ1FPM0tBY3IvSUVsQnAKNzhpdnZQemZGUnI5cTVQR2M1SmZmOTNJWTNYYVdnYkRERmZ5UVVLY1B2WmdZOGF6dHpERm5VSVhLazJ2RndJRApBUUFCb3lNd0lUQU9CZ05WSFE4QkFmOEVCQU1DQXFRd0R3WURWUjBUQVFIL0JBVXdBd0VCL3pBTkJna3Foa2lHCjl3MEJBUXNGQUFPQ0FRRUFHajFFMzgvdXM5eEVXMGJQcmlxdE5DTGlRVWlkbFA3eUVrQ3VxeVFsd1JNbHB3TE0KM0xzR0FBVis1aXNrQmJDQ3VhRVhaeDNjN1ZncGE3NFA4Vm9kUzZXNWxKeXA3bFNEWHdGYldNeXo3b1VEbi8rdgpaQjNhVlRRbVlKdXZKemE5RWR1UjNIUUMzV2swK1Iybjc5Ulc1L3B5VlBvMjN6eit6OTBJWld6eTlEdmlMa3pyCldNcGZKT2JRTjdHT1RpRW5DSDZaZG4vMnJJOVFpRXdmTWVUK2VYNjJ6UlFjSTg2SWR6U3pob3VzQVZ4eU14aTMKNnBoL2Vqd1JKenhFUmFWdDZCU1VxNXZZOGs4Um9rMHV5a0hpRHBTb2Y4VVE0Nnd2cERKQUNqMjNEYlpTaGU3QQp2Z3ZNWkdTVWIyZ3MxRjZvdHcyNldQSkx3UzJ0QTcxMWZtL3VBdz09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
            key: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb2dJQkFBS0NBUUVBcXVyREJkM3piSXduMU12bFZ2Y2Qzc3JJYWhueHdFOEd3czZqM05SV1B3V3N4WUVRCitKN0pGdEtlS0NEN0tLdTlYMEVRS0V1QWdFbjZYSm5lZkVReGJHNjY0bEpnQ1dINDNiM1RFaEFlT000N0plREUKak9wZFUwZk9nb0o5SkdSZmpCcHF1dW9MVkUrRmxMU2dmanY1S0k0TDU4VTg0enBRZ0Mrd1NINWE5V0NXL09zRApXZkZMeFRzSUR5aW40aGFQNDdCMXQvUnlxQ2NkYXROTldVWDFCSEdYRnpqWFVRRGJzekpGMHFLeWFrL1RvQkpqClJ5NGNFb0s5TnhSYXR3RFZxcUxtanBFY0pBQ1FPM0tBY3IvSUVsQnA3OGl2dlB6ZkZScjlxNVBHYzVKZmY5M0kKWTNYYVdnYkRERmZ5UVVLY1B2WmdZOGF6dHpERm5VSVhLazJ2RndJREFRQUJBb0lCQUdIL3M0ekVvMU5rMVYzTQoveFdySVdSaUx5R3UrSStFZ0dMb2Fzb3VzYmoyL3daTHA3aDJDdVRjSkxUcm5EYklxZzlZMWZQVXZyeFFMbzR6CjUzNm05eE91TmRlWTFkbnJZKzk0YlBLWmJVcXk3UFVkK0hTMzJJVHMwanJBcVJKWnZ2TXRIbTlIelBFdG0zRVYKVkVTdERJSzFWNXd2Mm1hTFJDY0xzTzhRREFWT3laWVRMVEFKSXp1ZndPdTJwR09ySFY5QW8yN01NL1BvRVk4NgpOK3h4T0tIdWt4ajl5cjFDcWk3MzBIcGM1NEp1UGlOaGFHTFhMM2lRcThOUUtOam1FdWM5T3lNaXViYmROWUp0CjgxRXh1KzBPL1MydW5TcnUxck5sRm5zZnpmMUJ3OVRMSG9nazVVQUxEVkUyb3hyV2FOVnpQM1lrZlNwQnE5R2sKNlY0eFZWRUNnWUVBMlcwem0xd3RMRGlOVTVQM2dreFdCQWJRSXBDVFN4NHNnNk1YVVBHUjNWOGk3Mk80VmUxdQpJSEVZRUhRYTBabWs0NWRmUER6MUtuRi9KRmZ0S1EyL1dHS2VxY2hQT0pHK0ZEZHY5b2U2NW0wVzVPMStZUzJGCmp6SllMRGpYYjRHL1YxaFh3b2FZaVJTUmxWdWlCalZoR2tscm1POTVBRWJSalcrVmJoRTV2bThDZ1lFQXlUMUEKSmdRUERkYVRaQWJ4aEFVcmlFaWNwSndyOW5kdGh3UWQyVnN1U0xkbWVwT3pma3V2Wnl2OUNsVVY5dG9TSnhPQQpRYTdBWkRNUU1zaW9mWjlESExXZjg4UnMrdXZSMTFDWVMvZ3BEbEZMQnNydWpnUkw4QysrNXRhclB1K3pVWGYzCnpKbXp4V1RTUExsYk1nbGdpNy82ODFHSUxjb2hUaWtvMUEwYWJka0NnWUFEbzdGT2MyK0tJdlF5cHZKb3F2QngKcHMrTEZKSnltbkd5VG1PUWhrcWp3dmpwdXVpVi82QnRTSmRYaHluL3dpdWhaNkkvZHVHL3NTQ29zaFFWTU5hNApHd1orM3d4OGJPd2FtSWIyUUhWZmNBb0hFUGFobDhwNWlDVXpzRXZpNFBBYi9TMlM0di9nbzRpQVVJVll4MEgxCjE2N0dZNVBKN21XSmtZbVZ5eElGWHdLQmdINUoyZ0FCZkJFTEgrUWVGMkxxZTY5RElrcUxWMnVNOTBkTDVnK0oKa1RwQkhpYSttbDRmMFN2R0J6NTh0eFA3Tm5rZlc5WGNmUkJrbXozRGZ6bVd3Tjg3ZSszV3p5Mkk0RjluVEt3ZwpTR09iMEtHcXFKcW5SNkNyMWZtM3JqQUk3VmVyR2U5a1pkVHl1SjB6RlhBSlFuVnhQV09GUHhpOVNMbHNTeHNJClBoWEJBb0dBTWFjdTZiZU5qVjY3SjEvK2o1aThZeEhHMnU5WXR1WHBLeTV4Rkt2RWU4L0ZEek1xNFRJa05yTGsKYklFTjBYcllpR3lmNUJhSW9FbWVXa3djREpPY21VbmtTMmQwKzVyaGJPcHlKQkc3TFdDL2N3WkVtQzMxY1FXZAp4UksraDd3UVE5aXZCUGhIekJwL1VpWVRjZk1rV3VDM3UxRXdFeS9uNFhvM2RZNGlMVWs9Ci0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg==
          etcdCAKeyPair:
            cert: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUN5RENDQWJDZ0F3SUJBZ0lCQURBTkJna3Foa2lHOXcwQkFRc0ZBREFWTVJNd0VRWURWUVFERXdwcmRXSmwKY201bGRHVnpNQjRYRFRFNU1EVXhOakE0TlRZeU1Wb1hEVEk1TURVeE16QTROVFl5TVZvd0ZURVRNQkVHQTFVRQpBeE1LYTNWaVpYSnVaWFJsY3pDQ0FTSXdEUVlKS29aSWh2Y05BUUVCQlFBRGdnRVBBRENDQVFvQ2dnRUJBT3dQCjBWRGtERWJJanRvS0lCTVdhOTFpYmtVWTFva0o4NW5nbUJhYTczVmE5eWRaTDh2Q1g0eHBGS1Z5MVdpUXZmL0EKVVVMdmNkV0VvUHArRGlZRkEzZW92RlVYSUVzekRBOWFveDE2bHFFb2FYcDhMOERTbDE4ZHZ5ZkNmQXdhd1ByVAphSzZ4cWxtYThQdEZmdTVMaGRuV2xMdmUzUnoxUUQxUHVHWnBOK3ZJVFhhUjRkRkZacjdMckdwOWFRWndLd0Q3CmtWN09uTmRENHdVNXZNRENPTVFmemNZSkFoZERiS3c2RUVVZlR3bTNGNE00TTRxNDQ2SUVnY3dENmswWWF5TG4KKzFIVjZETE4zZndKMXFXVStmTEFPZE5COHpzWEJONlNrRGlkQm5nVVlraW0rMVZ2UUJXSWJIN2RFdVFsS0dpRQozQjFiaW9qbUR3cjYyVWRDaFUwQ0F3RUFBYU1qTUNFd0RnWURWUjBQQVFIL0JBUURBZ0trTUE4R0ExVWRFd0VCCi93UUZNQU1CQWY4d0RRWUpLb1pJaHZjTkFRRUxCUUFEZ2dFQkFFSTVDQ2pnSTNvajZUWjBuTDRUcDZSZEtVZFAKMlBlWkZCMHNVWnhSZVNRRWJVdkp1dGNCb2JKM0cwcmNwWUR3OEdNZ3ppWE1hUFhzT0pzTEhIaTJvWVRWMFEraApjdjF3a2xaWEdoQ1NpUUcvUmhNMlMzbkthbDZCWUxWUVJmU2IrWVFHV24zK1ZqbUlXV0trU2h1VUFnMHorTlFnCjFjaWQ5eGZhY0w5aVdYVjkyRGZMRmd3MFFlcENxZElVbWFiaEdYcXRNR3ZSUGt1OGhkenpXQmgzOHJyNzlja0kKNi90OGo3cDlkUjhoWEFYWmhidXpXWk5UWmh0NDQzUzYzS1BiemFyNUlXODBsZnlBa0xqMW5HMkdkaHQ2TFZqRApPWFlveGRibjc1MFlVOGNWam5Pb3FsUlV2S1FVbWlSQTBrbllpQnBvNEhCUkZMSXVzSWhPYUVSWHhxTT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=
            key: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb3dJQkFBS0NBUUVBN0EvUlVPUU1Sc2lPMmdvZ0V4WnIzV0p1UlJqV2lRbnptZUNZRnBydmRWcjNKMWt2Cnk4SmZqR2tVcFhMVmFKQzkvOEJSUXU5eDFZU2crbjRPSmdVRGQ2aThWUmNnU3pNTUQxcWpIWHFXb1NocGVud3YKd05LWFh4Mi9KOEo4REJyQSt0Tm9yckdxV1pydyswVis3a3VGMmRhVXU5N2RIUFZBUFUrNFptazM2OGhOZHBIaAowVVZtdnN1c2FuMXBCbkFyQVB1UlhzNmMxMFBqQlRtOHdNSTR4Qi9OeGdrQ0YwTnNyRG9RUlI5UENiY1hnemd6CmlyampvZ1NCekFQcVRSaHJJdWY3VWRYb01zM2QvQW5XcFpUNThzQTUwMEh6T3hjRTNwS1FPSjBHZUJSaVNLYjcKVlc5QUZZaHNmdDBTNUNVb2FJVGNIVnVLaU9ZUEN2clpSMEtGVFFJREFRQUJBb0lCQUV4aExkUGpoY2xkV1VOWgpaVmxudHN6eDdVWDBMRzQ0eHhZeDRtUG1DN2JJRVJJdFBGYk1kSWdFOUFZNGFxNmpycVpTdnJoT3EyRnZ3WHByClVQNmlQcVgzOWIvK2RKZUFVOVdmK1FrdndnOHcrVGdGZUpvR1NhNGYvTnJMaGNHUTRvSUY1MmdtMmp5VjhvVUsKem5BaUJaUWZadzZHcExxYTdBY3FoVHExcnI5ZDkvb1Y1ZEdMMC93ZmZHcys1ZmF2ZUNSQ09SRVo4WE9qdTBOeApOcE1JRmZMeExVVUoxeFNTa0VmbFhVOENJUDkybWxuTEFUSjQ4cnZXc0tGVTg2NGtHWmttVkUwdVY4eE5yQXNRCmhOaEhaNGNWTU1lQjhmaXZIQ0Z2c0luZW9QZENaVW84VFlpa1loN0ZWQm16aDU0cGQxaTZmaVN5elkrRjltclAKNVRZOGJDRUNnWUVBKzdVbGpQUUZmc1FFeEpIN0k0WktVYWtLSE84aEJlTTJvK2FKR25ZUmhrQUcyc3dVNFZYcQo2REdTZFhRZmE4SHFPRm5WZWdkMVRMbzFObk1hZHZ5dlVYV1NlQTNSRk55KzlXR2wrNGZoalJLc1p6WklvZmN2CkVScHkrb2RTMStJRU8xZGJRT2hTMFFNZVZoZXlFVS9TcDJHMnh2SDl2SEpLTFAramVPMEx2bWtDZ1lFQThCWmUKSDRpN3c4K0QzQjFJWkk1dnhaSmsrVTI2MEloZnNSSS8yZTU0dDRCdEJUUVUvUHFsa1JibkkvZ1BzTmlSWkdQdQoveUU3aU5mdDIwYmhaZG9kRWU5K0xoTTV3STR3dFRCZm9iQmhXSW9EM1VibVA3MkRPNGdBL2pYbjBQUEhUSjBWCkJKT3lPTWVMOE5JUVA5ZDFCb3dtS29ZSVVNNkhic1A1N0FIR08wVUNnWUFvaGU1b3RmU1lod2haZVVNY1lnZGYKQkQ3cmo1Z2FjWTBmY0FNTXJvdDl1SnNoNkk5SUErUVF3OEpYaEgxMmhMNm5tZGJqa1lYUjkzeVBxcEpOSzdzeAoreUs4ajBUay9mRUVZbGN0bXArMmJaWXgrNnhQM3hMRnh5TlJzQzJuTWUwS2ZMTGxUVUhnQW9lRXhzWFZRYXVoClpQcmZKcUI2RVZCZDFENUdQcXVRV1FLQmdCRWF0dGZ1U05vNVpYNkFBNUZPYUg0ZDROMjJBUElzVlF6cmJPc2MKeHpMamptREpoaGxEemhuWkZOeUdKckFGcmM0R0pKZStpVnhGYmVlcVZCS0tpSG1ubzBpckMvbEE2QTF3aGMraQpIajFOajlycTJ5cDlXT1ViMmw1Qi91THZDeXJWSWhNeVpvY1BDRlErMHZPSmFRZnZZaVN6YWRJLzlId2FzQ3AxCk1lYzFBb0dCQUplQTFrb2g5QTliSXJKendFUDZlbWtnV1RWQzFRUTRCc2J5RnJsUnZSR3FxTWVlZlo5RkZlQUUKWTMyMFVqZDh5ODVrRUZIcGRUZDdOdkNvTFFsRVBtZnhvNlhMZ3pEOEZtVFFZZG5mN29CdDNYS0tSS1RmaXl4MwpCQnNySDdvSkpFem1xTi9WaVorZGNoV29nTXVtNFg0N2V4VzRrWUpFN2tscVhWYTJ0aG1HCi0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg==
          frontProxyCAKeyPair:
            cert: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUN1RENDQWFDZ0F3SUJBZ0lCQURBTkJna3Foa2lHOXcwQkFRc0ZBREFOTVFzd0NRWURWUVFERXdKallUQWUKRncweE9UQTFNVFl3T0RVMk1qQmFGdzB5T1RBMU1UTXdPRFUyTWpCYU1BMHhDekFKQmdOVkJBTVRBbU5oTUlJQgpJakFOQmdrcWhraUc5dzBCQVFFRkFBT0NBUThBTUlJQkNnS0NBUUVBOU9BYmhNU05zYi9DWlRBQXZYb25NdE9ICjJDNThlMjNReURQbHpud2VkMWlmNkhXVEMzYXE5U3JqODJSRzdkYW13YldWalJqYkN2NytmeVJPeHllZFlvbE0KK08wTTFCS1kydFlJc0xMdm5ONkhtRUltL1BodjFhOWZ2V1RJclBOWnZiWnZkbmtCakVNTzFWWUx4bm42dStEdwpUaFFveGFoUkZkbGZwTVZQdmNmVWk2Wmh3eHJSc0tKQ3lOdmpyaTNBUXZQam9jQXB4Yzdud3RlNjdDQm9qVFRVCkV1M1QwRHJQbzdNaVh3U0t3MGJlS2o3UTM3YmJ1VUszejFibmNSZHRseEtSdEJwM2lkbzY5bVMzWnJlRUNoSE0KVGhPb1cvSlNEdjNhcmFtRDdlb0NFQ1BYTG9XZmVhV2Naalk0YVFkZDVEZzZHalFISVJsa2dpaWZUNEdBR1FJRApBUUFCb3lNd0lUQU9CZ05WSFE4QkFmOEVCQU1DQXFRd0R3WURWUjBUQVFIL0JBVXdBd0VCL3pBTkJna3Foa2lHCjl3MEJBUXNGQUFPQ0FRRUFsNkpUOW5rQVU3RTJyNUJhVGQxRnpYMGZMMGJkYzd1TFFuUXRIRXpBRzR3d1F2UEEKVEhsV2l6NkFVZkF2b0F1a2hsYTlLcDdDblFwdTl2SHRGZnh1RzVjT3gydGZncE1NRnM5QUcxQXRjQkdNZXhHUQpkbVNMN1c0dlRMMDRHQ2o4WlAweGxZTldzNU5TUzloQVJXcjdkSTJyQTI3amVGRjNMeWo3aHltdTA2Yk0zZVh4CkhpT1k3VjFuT3pCcG9qdFlUU2hhSjFZZThsNlREOGd6eGJSdjBUVlNjaUNnTFVMVU1IVkoyMkc2dWxBZGN6VFMKQ295bllOTTZjOGtYWmRINko5VmNOUUpNemxhMi84cGdHTGdrdTZrRC9HUFFpcUYzQjJDRGxzcThVRURpY0luMwp1anJhNVBTeUJkR3orM0lnRVZCWGNOTm45T0h3VXR1dUx0QjJ2Zz09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
            key: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFcEFJQkFBS0NBUUVBOU9BYmhNU05zYi9DWlRBQXZYb25NdE9IMkM1OGUyM1F5RFBsem53ZWQxaWY2SFdUCkMzYXE5U3JqODJSRzdkYW13YldWalJqYkN2NytmeVJPeHllZFlvbE0rTzBNMUJLWTJ0WUlzTEx2bk42SG1FSW0KL1BodjFhOWZ2V1RJclBOWnZiWnZkbmtCakVNTzFWWUx4bm42dStEd1RoUW94YWhSRmRsZnBNVlB2Y2ZVaTZaaAp3eHJSc0tKQ3lOdmpyaTNBUXZQam9jQXB4Yzdud3RlNjdDQm9qVFRVRXUzVDBEclBvN01pWHdTS3cwYmVLajdRCjM3YmJ1VUszejFibmNSZHRseEtSdEJwM2lkbzY5bVMzWnJlRUNoSE1UaE9vVy9KU0R2M2FyYW1EN2VvQ0VDUFgKTG9XZmVhV2Naalk0YVFkZDVEZzZHalFISVJsa2dpaWZUNEdBR1FJREFRQUJBb0lCQUE0TWpZZWdmMENqYjlPKwpYVXlCcFo0VTVNWlNaSEh3YXZvUmdDM0lrVGJucnNUM2VlZytkckI2TlhuUlZ3QnVRRUw4MkFld1ZXRGNJRjNMCkIzb0ZtOUg3YnA4WmVwTHdQalRQOHMrd2RHN2JsWXYydjZYajJ3YTNlUmEwb1k3S3AydHEvQm9mOXliRThpVHIKT3VHNFBkcHBuVi9kYURsdTNyZ1NNSnFUZDIrNzJvc2JQcExHeHF0bSsvTzFiYWs0dmRMb3BqUGU1OHRsbUhZMQpmV1FNdXg4YSsxSHd5VTZabGQwVW9mcWZuYUxzZEZkSUpDTG1OWVNUMVE2Um41Tk40dTdPMkZkeDRQUHFNeWJICkdrUUtkUmJTdlQ1TzZONHhDQjZwZVpVM0d1eVpSRmFkNmJ1aDhlYm9wTnFOc2RTRlI5SnpkYW40dDNNRUkrSVEKTEhsS29BRUNnWUVBK2Z2dHFCZGJWUHM2NExIUXNwVFN2WVNpOTVYZ0JQMFdXalZzYmJudy9ybkhxWHAzSlNMQwowTGZ6QVcxSWwrL01hZHRaS2g4YzBlMkF1UTFOdnNkUklZTlNKNTIvMUdxSDhoNzNZUlJBYklPTi9pMjVnY3gzCld3RzI5YTFSSUdoTEc0aVN4NUdHL3kwcTZCalQ0MkJzRXdKR0d2QUt3Wnl6VUVaNlNWdmVYVUVDZ1lFQStzUzAKekZNbFBuZDRUKzZrc3RNelVHSmVkcUhjRk5UaHI1dTRackR6amc3dEFGSHJrVTUvVVVJdlRyK0QxQ2FzRWdMaApteHRuV3NOKzQxeEZNemluWHRKRTFHcWczN3R1SUhmMFdOVGQ5YXNGS1krNm5YK09KTk51bzdxVVMwM0hKS0dPCjI4TnFwQTJxN0xzOGMyS0tISnBlZ1dIVU5HREFCcjBKOFJZRGROa0NnWUVBZ01JK1d3SU16T3pLR3NuNzBML08KL0ViQkdmMWNjYlZhT2dTaVlMSVJhMktOY01IZmRJVS9DdnAwZEJ1eDlIQlRQWUw1bmpTQVI3Q3BTS2VOaitKaAo2MzBVWjh0Yzd6QWY3Wm45bVVjeEY3TjdBNXpSbkFXUXhKTlJoYUZMMUFGa0RqNStPOFM5WDlvSDY1dytKek9XCjl3T0kwSDhyU3laSFJlWEhQdG5PNHdFQ2dZQk5jQjBjMDdnMm1CSVJMUEt6UGtFa1c1d3NLa09hTXpzV1RaSWUKTkJxaURiM21VV1hiVTFCQnVaeCtSdW5ndzZoelQyeDN4M2lkTUsyb2JELzZWMDVvRzZxaHBlUFQ4ejlJeTRJcgpSR0Rla2xkZnhFQ3Vqa3RJMS9uY21hdGRyY0VIY05SNnpOZkxuV1RoQWRqakVOVHhqRUlPMWpUL2o3ajgyN2VNCk9XNEwwUUtCZ1FDKzF0N3VzdmFYeGNHSHRKVTRpelVZUXIvbkxtSkVOZHVPZmJOZkx1d1RhaUtIQ0ZKMFppLzYKc3hadGlSaTRnRitwTzhzVkxJLytKb2xMMUc0S2lYR3ZnVDdDRHFJYVVUcEN4Ky9RNUo4bUZqSUxCd2NlKzlHUQpMVVN0T015VXpmUm1OODl5TXBnSnRwWGU1ZWxaZU1pa0Y2RGF2M2pOQmhWQjdGU3dVSFp4WUE9PQotLS0tLUVORCBSU0EgUFJJVkFURSBLRVktLS0tLQo=
          saKeyPair:
            cert: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUM1RENDQWN5Z0F3SUJBZ0lCQURBTkJna3Foa2lHOXcwQkFRc0ZBREFqTVNFd0h3WURWUVFERXhoellTMWoKWlhKMGFXWnBZMkYwWlMxaGRYUm9iM0pwZEhrd0hoY05NVGt3TlRFMk1EZzFOakl3V2hjTk1qa3dOVEV6TURnMQpOakl3V2pBak1TRXdId1lEVlFRREV4aHpZUzFqWlhKMGFXWnBZMkYwWlMxaGRYUm9iM0pwZEhrd2dnRWlNQTBHCkNTcUdTSWIzRFFFQkFRVUFBNElCRHdBd2dnRUtBb0lCQVFEZSs4dlZtaUZZYU5uWm5pTVgwb2Q3SkVFSDFxcUcKU0o4VnFUZ0VraitFd3hidkNZbDlOd1g5ZUNPcUxzMEo2MlBUTEpiS0ZRUVJmTEp0M0lNV05LSnJlNUVJQmR2dQpjMyszbVRlQ0NQQjlhT2VrbUZwU3VRU0FSb1AwRXBlZTFuYitRcFZrclc1MEkvM2NqZk9WWWdZNjVvUXVuTlB4CnVQNzdoQlo3TnhBY3lDTUIzeDJSWnd2TTRvU3htY0VDL21rSStxYWJUT3Y0ZVFUSzcyVFJZaEIrY2cvTzVlUkwKeExEZUNEUmdQWllLTEhhQkw1NVNuU1Mwb3dBaDBScE9BZUlRMjBib2Vma2xnUDFwNkpMY2xjM0lLUHJVeTJZSgpMVWpjbFBoOUwyU3FObkU3MzRJOVlQK0U4QlIrWXEwMWhJWURVZDdzREJ1WnJNSTU5OXNVaXJXeEFnTUJBQUdqCkl6QWhNQTRHQTFVZER3RUIvd1FFQXdJQ3BEQVBCZ05WSFJNQkFmOEVCVEFEQVFIL01BMEdDU3FHU0liM0RRRUIKQ3dVQUE0SUJBUURXKzk5ZHhzTjhEc051MmNPZkI1ZWI5L3QxTWJuclUwQXg0YVRaM2Y4ODRRU2tlaWd2V3g2Two0dWZtQnJ6bmMvRVM5UUxGcDlCSWg2V0lHcGhxSXdsb1lXcnNSTEZUKzhHT2t0VldsYWY5VjlLbFdFNFJpQVMzCjJTTk90MzVPRGYxSXBJTGQ2eTYwLzlZK2MrcUczbkJBR25IT0YxNG9YMUpHSWJ1cnRqejEwM0NwZWY1T2NjSjYKc0N1UG9nQnBIcjhVWHB4K0tsak5Wd01yTk1wRjVKZ2pYdHNIV295SFBKaUVMeEVGdGduOG9ITlYxVmxkOGZzTwpGKzlPTkxZeGpWNmhnU1NFalpNWCtmbzY0cTN6a2s4aTNkS1VoOUhRbUk3VERydm45V0RWMnZJUEgxMEtDSlc0CkZxbmhQdExSNmZreGJFMmJYbGpUbkkwMFZ5aS9OdTRSCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
            key: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFcEFJQkFBS0NBUUVBM3Z2TDFab2hXR2paMlo0akY5S0hleVJCQjlhcWhraWZGYWs0QkpJL2hNTVc3d21KCmZUY0YvWGdqcWk3TkNldGoweXlXeWhVRUVYeXliZHlERmpTaWEzdVJDQVhiN25OL3Q1azNnZ2p3ZldqbnBKaGEKVXJrRWdFYUQ5QktYbnRaMi9rS1ZaSzF1ZENQOTNJM3psV0lHT3VhRUxwelQ4YmorKzRRV2V6Y1FITWdqQWQ4ZAprV2NMek9LRXNabkJBdjVwQ1BxbW0wenIrSGtFeXU5azBXSVFmbklQenVYa1M4U3czZ2cwWUQyV0NpeDJnUytlClVwMGt0S01BSWRFYVRnSGlFTnRHNkhuNUpZRDlhZWlTM0pYTnlDajYxTXRtQ1MxSTNKVDRmUzlrcWpaeE85K0MKUFdEL2hQQVVmbUt0TllTR0ExSGU3QXdibWF6Q09mZmJGSXExc1FJREFRQUJBb0lCQURyM2xYRFlRS2N6bXlDcwpiQlZackZCSlJ4VStsSHZNYXAxODBYNkwrbFl1alRzTFo1dUFXSW91SWloWUZncmdmOWFSZlJMVnVleXg4REdUClZlc2lZaTFRVVFzeEdYY1dmaGpjWU14M2RybURhM2FnZjRwT3VUeis2Tmc3cm5MbnZqSUJBNmJMSW1GK1B4ejcKUVFZcEZRS3dnUlllalFIb2JTMndRWTIyQXdISkFMV2VCMDRIOFJFdnNwWkcrK3VxeFFSY0JNaXdvakJvU0syTgpGQUU2NDlXeFFYWlpTZEg5UVJCa3VRN3Z5S3VaMzhQdjNuOHZkQXVpTndUWE8wbE44SExRM3RHOXNsYW1wa1FTCmI2dU55UitYRXowa2c4WEVGYWk0b21sTUpCUDJPSlRvam1KYWJJblpJc3ByU3FialFpVDg1aW9KeGxMS3NUU0IKeGs1N2h0RUNnWUVBL2lMaS9oNDhoSi9HcWlMYktNamMrUU56bHRYdGVsOERXSHdLUEN1K0REVVZqNWVNY1dDTwowNm1Ec3RKamJUUmExK0lDdVlsWThNczh1YnIvMU1UdEFLM2dVT3gwSXN5d2Z2RVdNY3h1TVpZalVTczhLWE45CjNyYWJ3Vm1qRjlhUUNHUmZaNHdPSWNiMlVPOVkyNnN6QllqV0M5MGt2SDhEU2JBd0hSRjgxNlVDZ1lFQTRKNXMKZVZFc2hVeGlDUCtwZTgvNWRPRWhlbEh4d1NSUklMajhVNXlwVGN0WXd1M0szQjJGSS9NZzZXNHJsWk5FTlpZZwpRZ1YxdWk1SjQ4Mk1MMHYvemplYmVHZkJwdlNxYmdra0dPdXRoK0VoeWlmaStkZ2REaHlhclhqYnc0TVJrUk9oCnl3aG5Ja2JLbnBmTDVZSWk3WUJOaTBHWmVqN0t5M1g4c3lhNnFCMENnWUVBbTFURmUxOE56RjVBTmxOeHN2NzYKbVRVejNGakxld1ZCN1Q1N253VjVkc3FuY0FuSUxMQkEvRHhiSTl5V0t2UmFKaU9kV0x3TFlicEhWcHBtcml6agpVNHZ6VkdNQ0pSY0pOYjJ6dkNKZ28reEpqOTRtT291OXZuZk13YVJCSEZ2bjAwbE9TdUwrN0VSSTMzVTcwYUJWCllpZWQ5TWhwSU5GZE9CZjVnSHJrM0lVQ2dZRUF2eGlKR0wxaXJ4Vkk0bmIvN2dJR0xOWEw4WkQ5cUdBSlZWbUwKOG1aNTgyRm81bzMvNUR5SkpRaVhtNERMTzduS2NmeUMvU2cvL0dHZEkxMmdRaXphT01zK1RiV1lIejVRTU1VKwpIS1dGUVBEY0dtek13ZHFHL1phQVVjMWN5bjBiMk4xbTdLRDlmVC9VNmhBaXUrTjNhNitZU1QxS0lhS0NUWTdYCnFtTHNzbEVDZ1lBTVgyRy9CaWx5T2kvSGRnQmVMUWQ2bnQvbFJqODZianZ5bGhWSHVIQitFSkpjazZDZjFRR3EKbkJNTHdscllPdERjczg4S25pcVd3YVNXeExPeE01cFhtOVg5SHI1QWovamZqVGZEY2Q5dnBuK0phUXlSalNlQwpzdlRoaE5PTit5SC9POGJDZFJtSTY0NUFybG5IYStlNzNxYlVmZFlFUktRTUNNWmREbGtHSHc9PQotLS0tLUVORCBSU0EgUFJJVkFURSBLRVktLS0tLQo=
          adminKubeconfig: |
            apiVersion: v1
            clusters:
            - cluster:
                certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUN1RENDQWFDZ0F3SUJBZ0lCQURBTkJna3Foa2lHOXcwQkFRc0ZBREFOTVFzd0NRWURWUVFERXdKallUQWUKRncweE9UQTFNVFl3T0RVMk1qQmFGdzB5T1RBMU1UTXdPRFUyTWpCYU1BMHhDekFKQmdOVkJBTVRBbU5oTUlJQgpJakFOQmdrcWhraUc5dzBCQVFFRkFBT0NBUThBTUlJQkNnS0NBUUVBcXVyREJkM3piSXduMU12bFZ2Y2Qzc3JJCmFobnh3RThHd3M2ajNOUldQd1dzeFlFUStKN0pGdEtlS0NEN0tLdTlYMEVRS0V1QWdFbjZYSm5lZkVReGJHNjYKNGxKZ0NXSDQzYjNURWhBZU9NNDdKZURFak9wZFUwZk9nb0o5SkdSZmpCcHF1dW9MVkUrRmxMU2dmanY1S0k0TAo1OFU4NHpwUWdDK3dTSDVhOVdDVy9Pc0RXZkZMeFRzSUR5aW40aGFQNDdCMXQvUnlxQ2NkYXROTldVWDFCSEdYCkZ6alhVUURic3pKRjBxS3lhay9Ub0JKalJ5NGNFb0s5TnhSYXR3RFZxcUxtanBFY0pBQ1FPM0tBY3IvSUVsQnAKNzhpdnZQemZGUnI5cTVQR2M1SmZmOTNJWTNYYVdnYkRERmZ5UVVLY1B2WmdZOGF6dHpERm5VSVhLazJ2RndJRApBUUFCb3lNd0lUQU9CZ05WSFE4QkFmOEVCQU1DQXFRd0R3WURWUjBUQVFIL0JBVXdBd0VCL3pBTkJna3Foa2lHCjl3MEJBUXNGQUFPQ0FRRUFHajFFMzgvdXM5eEVXMGJQcmlxdE5DTGlRVWlkbFA3eUVrQ3VxeVFsd1JNbHB3TE0KM0xzR0FBVis1aXNrQmJDQ3VhRVhaeDNjN1ZncGE3NFA4Vm9kUzZXNWxKeXA3bFNEWHdGYldNeXo3b1VEbi8rdgpaQjNhVlRRbVlKdXZKemE5RWR1UjNIUUMzV2swK1Iybjc5Ulc1L3B5VlBvMjN6eit6OTBJWld6eTlEdmlMa3pyCldNcGZKT2JRTjdHT1RpRW5DSDZaZG4vMnJJOVFpRXdmTWVUK2VYNjJ6UlFjSTg2SWR6U3pob3VzQVZ4eU14aTMKNnBoL2Vqd1JKenhFUmFWdDZCU1VxNXZZOGs4Um9rMHV5a0hpRHBTb2Y4VVE0Nnd2cERKQUNqMjNEYlpTaGU3QQp2Z3ZNWkdTVWIyZ3MxRjZvdHcyNldQSkx3UzJ0QTcxMWZtL3VBdz09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
                server: https://az1-d30d08c.eastus2.cloudapp.azure.com:6443
              name: az1.pharmer
            contexts:
            - context:
                cluster: az1.pharmer
                user: cluster-admin@az1.pharmer
              name: cluster-admin@az1.pharmer
            current-context: cluster-admin@az1.pharmer
            kind: Config
            preferences:
              colors: true
            users:
            - name: cluster-admin@az1.pharmer
              user:
                client-certificate-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUM1ekNDQWMrZ0F3SUJBZ0lJVkNUdXpoQVhDNFV3RFFZSktvWklodmNOQVFFTEJRQXdEVEVMTUFrR0ExVUUKQXhNQ1kyRXdIaGNOTVRrd05URTJNRGcxTmpJd1doY05NakF3TlRFMU1Ea3dNelUyV2pBeE1SY3dGUVlEVlFRSwpFdzV6ZVhOMFpXMDZiV0Z6ZEdWeWN6RVdNQlFHQTFVRUF4TU5ZMngxYzNSbGNpMWhaRzFwYmpDQ0FTSXdEUVlKCktvWklodmNOQVFFQkJRQURnZ0VQQURDQ0FRb0NnZ0VCQU1HNXBnajRGMmtvVnAzSG5pWEVER0tkaDMyTVRZU20KQzJtbW1DcHVhUGJsNHhzcHpVaTErYm54NXZ0OGxFbExtT01xRzlBTHhiQXNsNXVTUk5FaVpidWxpZjVMSitzWgpMRGM5dFo2aUdtZm9RVUNicjJnM1JSZkExMEZxRW5mLy9RN0l2a0kxbUZPeHhrYUlNWTdUK01oU2dJY0Z1WFVBCktyQjJuTGZOL3BVUXVxc2VjUVJoOWg5eEI5QnU5U2RKN3g2cE9OUU82YTZhVHVycE9ydXk0akk3dzcvSm53a3kKQ0h1YzBSck81T1J5VURPb0FiTlpUWGtGOTJlekk5V0c5anZvWWtUbmVtVkdBZGo1eExVOXFEdGFyT2ljZ2lVRgpVMitHY09manZiV2NvWWorN0pQaDRaQitqblh4UkV1MUFzYVA0eDArUkJmQnlURjNWd0RJbWYwQ0F3RUFBYU1uCk1DVXdEZ1lEVlIwUEFRSC9CQVFEQWdXZ01CTUdBMVVkSlFRTU1Bb0dDQ3NHQVFVRkJ3TUNNQTBHQ1NxR1NJYjMKRFFFQkN3VUFBNElCQVFDTE9peUlmcUpIMVZic01LNjdjQ29rMFdKOGFjTXBuYkMrcWIrZUcwQlFLSFluR0h2YQpKSHZESWp6U0IreXZCNGFyYTgzbThlTy82K1poa04xMEd1U0MwVG80RGJ5L1NwTlJQaDZUVEp5SlFkT2RvaWtqCmswYWMxajlGOEpEYUxMSG9RUXFIeTlBMHhMckhsUnl5QzF1NTQvRUFPK3lLNU5rMk13L0Q0UERKcEVOMytMUGUKZ2RSNnFSNHMzQjBaOHdLVmVCOXBiZlVjWFhkcGhVQlQxUDhkRVpvMGlFa01hQzdZR3BwV1QzUW81MGNNa1lIeQpFTUEwRGs2Zkk2MzZjL3JCYU9UNVVXSVBCSzEwYlZvRjZHUUdmdlVhamN6WktteGNORTF3QWVnWU9HUUFaVmQvCityOTBDb2dYSlEvcWRWd1h4Kzd6bC9xSDJCQjQxUFhTUWdZUwotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==
                client-key-data: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb3dJQkFBS0NBUUVBd2JtbUNQZ1hhU2hXbmNlZUpjUU1ZcDJIZll4TmhLWUxhYWFZS201bzl1WGpHeW5OClNMWDV1ZkhtKzN5VVNVdVk0eW9iMEF2RnNDeVhtNUpFMFNKbHU2V0ova3NuNnhrc056MjFucUlhWitoQlFKdXYKYURkRkY4RFhRV29TZC8vOURzaStRaldZVTdIR1JvZ3hqdFA0eUZLQWh3VzVkUUFxc0hhY3Q4MytsUkM2cXg1eApCR0gySDNFSDBHNzFKMG52SHFrNDFBN3BycHBPNnVrNnU3TGlNanZEdjhtZkNUSUllNXpSR3M3azVISlFNNmdCCnMxbE5lUVgzWjdNajFZYjJPK2hpUk9kNlpVWUIyUG5FdFQyb08xcXM2SnlDSlFWVGI0Wnc1K085dFp5aGlQN3MKaytIaGtINk9kZkZFUzdVQ3hvL2pIVDVFRjhISk1YZFhBTWlaL1FJREFRQUJBb0lCQUJVQlRqRjJ4UU1QN3FSWQorcHJac1FZWVVwS1lYZWRlSWFxbzk2TFNLZXRyYmI2S1A4bjhnVUZhSzFObFpLYTEzYlB6NHVRaUFxTmhrbWE1CjYyQkp2SUltSnRvOXgreEQ1SGx3NVhwMzFTa2pFOEF2b1V0Smd1SmFkSHlSUmNOaExFMG9Fd2tXeXBkNGxTa2MKcDFMM1JPaGptYkFLUE51azB2d2pRRWJsdlE3b2V5WEJJczVSNFJ4WmtpdzNDVDBHaW1Nak5YTGs3eTVDWnpUZQp3aUhuazhNeVNEZWs5K1pxSUM3MS9qSHVaZCtYMXR3YXRUckIyQmtlNVp1dkhYSUhnNCtqbEJOTnEyaGphWjhjCnVUVlUvQmhvaiswamVuZlJBSmtwZ3RGQm45U2tSZytRSGZ5a1MzTk84OHVWNytSNmEvbVE3YzNTVFplbGN6YXMKYno3T0dNRUNnWUVBMlFDcUlTRG9GT2JSRUw0aXVFNlJsaDNRR1IwQXpUUmsvbDhrN0RaN21uTWN5SnRRQU1rRQpuRVFWQzIrSzBaMFNFL0FyRUhXL2F0VzJad1pkcS9TZ2JseENtbzdoRkdabTlVdVBGU3U1b1pnZlRlZDBuR2Y0CmhTR3lIb0ovbjdiaVdjQ1BLODJvVW5UMmYwVkVwbEp5ZDNFSHBiNzZ1Q0JpeHA4ZHBEcmx0eGtDZ1lFQTVJb1gKeHdLQmR6bUpkMVRVZ2Q2MVY3Zk5XN3NCRVRHOU9UbzBtVzRyZWN0d3RYaVd2bFlDR1JvZ2V2UTdwdnc4ZXowcAp2eHlpZ0M0dzEvcVAzSkRSQUlUU1B4R3ZIanJjWWZ5bSswTDRWcENlYmlWcldpQ2xDVTc2Y0NBMG1JdW9QZjFqCllGZDRtZHVJOU15L2czR1RPZGdPdGdjbVBpYjF0aDdLd0NGRmlvVUNnWUJsT01RakUrQnQ3NFRSMUg4SmpjeW4KUTF6UWVoRG5wMnI4cEpEcWhqZDl3ZmhKTXZsTWhIZmNGSDNraWJFdE9hRTNINjVXelRYdXRhV1J4UXhvcTRFeQpPV0x3Q1huQU5Fd09XNkp3YzZieFU2NDJmcUtNV29zNFBwR0JYY24rVENhbFN3YjluYjdJUjdxN2Z5b1lreGpGCjBqbUkvTjZpaUtqS2tXb1lNMGozOFFLQmdBSEhTVlFWSXVqSW05Yy80NzkyK2paS2llQ1MzZmxDUjlTd0xVU2MKWk40M3hSTEVjM2ZidDN4MmhaWXZYRjk2U3dZeWZhYXRGRjZtL1Y1UTV0bXlqczNRT3NxZTJwd0RuVDl0OWVheQpxTGdUdkFmalpxaDI1SkhqK1hMeDdUYmFyMTU4SUUvWm5taWtHcmE4K1NpT3M3U1poOVBHem5kaFdqd21sVEZWCmtQTDVBb0dCQU1DZUxnM1NXcW1SMUR0OVgvZkE1RGp4bFIxeXdCSjZoSXNJN25sRHBrYW5yZEp1N290bzRYZmcKOHUzSERXdzlDMXJ3R3U5aTN0enlqRms2U2Z4dFNXV0Joc0RCTzRKSFF6TUdXTEx1amZjNUlzMG43d1p1ZFA3Ngo0Y3l2WXlHaEMwNUVVRGJ5VEo4WDlQSzNQZ0R3a2NveVNkbFRoMzdxbCtiUFlwUERHUnZPCi0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg==
          discoveryHashes:
          - sha256:1e578d85dc78ef9e8f8d5ac9cfee7d066c97ff42eaa40d80de1b46a409f76c63
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
      - host: az1-d30d08c.eastus2.cloudapp.azure.com
        port: 6443
      providerStatus:
        bastion:
          image:
            offer: ''
            publisher: ''
            sku: ''
            version: ''
          osDisk:
            diskSizeGB: 0
            managedDisk:
              storageAccountType: ''
            osType: ''
        metadata:
          creationTimestamp: 
        network:
          apiServerIp:
            dnsName: az1-d30d08c.eastus2.cloudapp.azure.com
            name: az1-d30d08c
          apiServerLb:
            backendPool: {}
            frontendIpConfig: {}
          bastionIP:
            dnsName: az1-bastion-d30d08c.eastus2.cloudapp.azure.com
            name: az1-bastion-d30d08c
  config:
    masterCount: 3
    cloud:
      cloudProvider: azure
      region: eastus2
      zone: eastus2
      networkProvider: calico
      ccmCredentialName: azure
      sshKeyName: az1-sshkey
      azure:
        rootPassword: QpNqy8m14iqzEand
        vpcCIDR: 10.0.0.0/8
        controlPlaneSubnetCIDR: 10.0.0.0/16
        nodeSubnetCIDR: 10.1.0.0/16
        internalLBIPAddress: 10.0.0.100
        azureDNSZone: cloudapp.azure.com
        resourceGroup: az1
        azureStorageAccountName: k8saz1by5v4x
        subscriptionID: 1bfc9f66-316d-433e-b13d-c55589f642ca
    kubernetesVersion: v1.13.5
    caCertName: ca
    frontProxyCACertName: front-proxy-ca
    credentialName: azure
    apiServerExtraArgs:
      cloud-config: "/etc/kubernetes/azure.json"
      cloud-provider: azure
      kubelet-preferred-address-types: ExternalDNS,ExternalIP,InternalDNS,InternalIP
status:
  phase: Ready
  cloud:
    loadBalancer:
      dns: az1-d30d08c.eastus2.cloudapp.azure.com
      ip: 52.232.224.99
      port: 6443
```


Here,

  - `status.phase`: is ready. So, you can use your cluster from local machine.
  - `status.clusterApi.status.apiEndpoints` is the cluster's apiserver address

To get the `kubectl` configuration file(kubeconfig) on your local filesystem run the following command.

```console
$ pharmer use cluster az1
```
If you don't have `kubectl` installed click [here](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

Now you can run `kubectl get nodes` and verify that your kubernetes v1.13.5 is running.


```console
$ kubectl get nodes
NAME                       STATUS   ROLES    AGE     VERSION
az1-master-0               Ready    master   24m     v1.13.5
az1-master-1               Ready    master   11m     v1.13.5
az1-master-2               Ready    master   17m     v1.13.5
standard-b2ms-pool-fpvhk   Ready    node     7m23s   v1.13.5
```




You can ssh to the nodes from bastion node.

First, ssh to bastion node
```console
$ cd ~/.pharmer/store.d/$USER/clusters/az1/ssh/
$ ssh-add id_az1-sshkey
Identity added: id_az1-sshkey (id_az1-sshkey)
$ ssh -A capi@34.205.72.251 #bastion-ip
```
Then you can ssh to any node in the cluster from bastion node using its private ip

```console
capi@az1-bastion:~$ ssh 10.0.0.4
capi@az1-master-0:~$
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
NAME                    AGE
az1-master-0        27m
az1-master-1        27m
az1-master-2        27m
standard-b2ms-pool-smnwg 27m

$ kubectl get machinesets
NAME                AGE
standard-b2ms-pool   27m
```



#### Deploy new master machines
You can create new master machine by the deploying the following yaml

```yaml
kind: Machine
apiVersion: cluster.k8s.io/v1alpha1
metadata:
  name: az1-master-3
  labels:
    cluster.k8s.io/cluster-name: az1
    node-role.kubernetes.io/master: ''
    set: controlplane
spec:
  providerSpec:
    value:
      kind: AzureMachineProviderSpec
      apiVersion: azureprovider/v1alpha1
      roles:
      - Master
      location: eastus2
      vmSize: Standard_B2ms
      image:
        publisher: Canonical
        offer: UbuntuServer
        sku: 16.04-LTS
        version: latest
      osDisk:
        osType: Linux
        managedDisk:
          storageAccountType: Premium_LRS
        diskSizeGB: 30
      sshPublicKey: c3NoLXJzYSBBQUFBQjNOemFDMXljMkVBQUFBREFRQUJBQUFCQVFDM0VudlByUDVBYkJWVnB4REVjUm9WSWJnL1VyV2xiSkNhVkNmei9EaittR0pVbjA3b0FLdXhOQ2JXTTF6a2JXTFh4V1ZGa3laeDZTNEhuK1VDK1JqL1k2VXFYclk5MS9weWhSZUkzNGl2WmNrUkZZdlRHcE9raXZ6ZFdMT0tjajZsWjh2SGF0MnVON2R3N284UWsvd09TNDJzRzBBUk83d0JPbG5GcEZ3VWkrWE41NXNoK210eENtREoxNTVqdGd1QWtnUENiRHpvYXdCZEs2bnRNSUU3bklpOVJ4ODlSVENwZk1lU3VsUnpVOGNvMmRQMVBtNndYWDRDYTkydzE1K3d3b2F1SFJKNTB2eGppdXNwTENhTngwU2hiWlRJdWFFUVZkMVdyTXEzaEQ0U3lBWTBlZUIzZXVueFE5Z2ZVS20zbHJlLzdtRTVyaGk2V3E0TDVyV3o=
      sshPrivateKey: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb3dJQkFBS0NBUUVBdHhKN3o2eitRR3dWVmFjUXhIRWFGU0c0UDFLMXBXeVFtbFFuOC93NC9waGlWSjlPCjZBQ3JzVFFtMWpOYzVHMWkxOFZsUlpNbWNla3VCNS9sQXZrWS8yT2xLbDYyUGRmNmNvVVhpTitJcjJYSkVSV0wKMHhxVHBJcjgzVml6aW5JK3BXZkx4MnJkcmplM2NPNlBFSlA4RGt1TnJCdEFFVHU4QVRwWnhhUmNGSXZsemVlYgpJZnByY1FwZ3lkZWVZN1lMZ0pJRHdtdzg2R3NBWFN1cDdUQ0JPNXlJdlVjZlBVVXdxWHpIa3JwVWMxUEhLTm5UCjlUNXVzRjErQW12ZHNOZWZzTUtHcmgwU2VkTDhZNHJyS1N3bWpjZEVvVzJVeUxtaEVGWGRWcXpLdDRRK0VzZ0cKTkhuZ2QzcnA4VVBZSDFDcHQ1YTN2KzVoT2E0WXVscXVDK2Exc3dJREFRQUJBb0lCQUJOSjNrT21UVytLTThFLwpoZk84bXV2cERwbVZaRkFXblRHMWRqUXR1ZStSTEtNUDJlZDEwcUVzQm4rQkQrTjladkdtK2FHWC9HLzZDb0NCCkowYmw2ZTFXbVZ0YWVVY1F6M0ZyZG14VWFQbFo5eEpXdTlHMU5pTWJCY05vaWhvbktWU1NHQlZkdkJlVUJUN2YKMDdFQ2RvY25ETGs2Y2NpZkM1THhpKzNZQUYrbHBCVk84enZ2WVZRUXllMHFWTTFEYlkwa09GZjBmK0xFbklxNgpUQXFTZi9waWlKcUpIM25lWUlqR21heVJCNGtPUS9OQmVuWnorUFVkKzU2RXJ0Z0dScTMwWW90eXBEejRxKzA5CkhtbzdXdEFIMmhVNWhLVGEzUUc0bkdBbEpVRmQ3ZFFxQzFTTzhrZWNMWENZdFUrWlY3NjBWb2ZnVDBlRE1GVFEKUmg0R0Jla0NnWUVBNUNLSXl1b1VyUXg3REZMaVBCTitKTm90bzgza2k3ZUxpclBsanY0bGwvd3pvZ0xYUVRpTwptZDF1dmk5U2d2U2RQcEVCWTZVRmxhamlLSS9oN011UkMyY3FCdjdqazZWazkvb2FsVjBWVXRCZk9LQ29wb2doCktWSDFCVTc2SU43RHlrQUpHQ0NhaGNFbldtekFzdXdlQWZJZkRFKzF0a2hHOVhJVFYweWgwNjBDZ1lFQXpXN20KaGZaVHFQdjMxKzdQcExFZmNFbHE0Y2l0TjdBRk1LY2hKb0dCRmFGWGc0T2RJb2hWWFZOb3ZkTDVWUkhMbDNwUgpkZW9IUTRxSzZqV0tPV01qcy9wS2NER0k2N3R4NEhQdDRlMFlGZ1BTMFVZbWpZeElkMGJ3SzYzcng5SXV4M1hFCmVReEREY3lFWXRCbUFoUTNZMTN6ditWRmtGMzZHM1dpc1hVbTJ0OENnWUJxYm80aEZLb0d2YzdlUmdEVUJFZ1MKaTFORm0zWG5sUDdkKytXNkcybVFpWkhSSU1BcDVtZm84cnlLcitzdnUwMXM5aHVPMEZ0Vm9nKzQycitOU0w5bgpjWDdTK3JGVG5aTUllYjlUTmJVUUNMU1Q1NmdtNFZXUFFIUXVRTlZDNW9xelhjS2daZjJSTHpiYjRlYlkwbjJCCmJPTDlUR2E3SHVjejlUOSt0L3E3bFFLQmdDaS8vdXBEMm9TQ3RyOFdtQW5MT0xsRlZ2WkNvRm1UaVBRRnN3VzQKV3FxM3ZteFFCek42WjdTRGZ4dG9aaDBCMHFqUmtxY1pMU2V3cTYyWndUbHcrUHdTZ2dHUFVlR3c1UDNwQVI3MwpzUGRzK3J5WWRiMU9QbkdxbUttUmJsdk16WXF6U2EzWlNOUEw1ZGJVRCtnSnFwTURaLzZBdERQVzhHM1IvOXZECnFWbHhBb0dCQU5WMFFFeTM0ZFNLbjdqSVV4NzdLM0k4S3hraFFmS2pzQXRiS3dsQmtkVFNJVjhER3RvZUlXcmUKK2hXWVA1RXk1RWJnYXhuRFpPblN2d05LSy80VGU4TmJPdHJuRVBRUjhGMUdrWVkyZjAyU1pmTDNiY1RZOThFeQo5ZERpbG1RbmFEOWp4R2hFTVducUNUUVRVeWZRVFlUdk1TTjNxRExPYkNLaHJoclR1a0N1Ci0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg==
  versions:
    kubelet: v1.13.5
    controlPlane: v1.13.5
```

 

#### Create new worker machines

You can create new worker machines by deploying the following yaml


```yaml
apiVersion: cluster.k8s.io/v1alpha1
  kind: Machine
  metadata:
    labels:
      cluster.k8s.io/cluster-name: az1
      cluster.pharmer.io/cluster: az1
      cluster.pharmer.io/mg: Standard_B2ms
      node-role.kubernetes.io/node: ""
      set: node
    name: worker-1
  spec:
    providerSpec:
      value:
        apiVersion: azureprovider/v1alpha1
        kind: AzureMachineProviderSpec
        image:
          offer: UbuntuServer
          publisher: Canonical
          sku: 16.04-LTS
          version: latest
        location: eastus2
        osDisk:
          diskSizeGB: 30
          managedDisk:
            storageAccountType: Premium_LRS
          osType: Linux
        sshPrivateKey: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb3dJQkFBS0NBUUVBdHhKN3o2eitRR3dWVmFjUXhIRWFGU0c0UDFLMXBXeVFtbFFuOC93NC9waGlWSjlPCjZBQ3JzVFFtMWpOYzVHMWkxOFZsUlpNbWNla3VCNS9sQXZrWS8yT2xLbDYyUGRmNmNvVVhpTitJcjJYSkVSV0wKMHhxVHBJcjgzVml6aW5JK3BXZkx4MnJkcmplM2NPNlBFSlA4RGt1TnJCdEFFVHU4QVRwWnhhUmNGSXZsemVlYgpJZnByY1FwZ3lkZWVZN1lMZ0pJRHdtdzg2R3NBWFN1cDdUQ0JPNXlJdlVjZlBVVXdxWHpIa3JwVWMxUEhLTm5UCjlUNXVzRjErQW12ZHNOZWZzTUtHcmgwU2VkTDhZNHJyS1N3bWpjZEVvVzJVeUxtaEVGWGRWcXpLdDRRK0VzZ0cKTkhuZ2QzcnA4VVBZSDFDcHQ1YTN2KzVoT2E0WXVscXVDK2Exc3dJREFRQUJBb0lCQUJOSjNrT21UVytLTThFLwpoZk84bXV2cERwbVZaRkFXblRHMWRqUXR1ZStSTEtNUDJlZDEwcUVzQm4rQkQrTjladkdtK2FHWC9HLzZDb0NCCkowYmw2ZTFXbVZ0YWVVY1F6M0ZyZG14VWFQbFo5eEpXdTlHMU5pTWJCY05vaWhvbktWU1NHQlZkdkJlVUJUN2YKMDdFQ2RvY25ETGs2Y2NpZkM1THhpKzNZQUYrbHBCVk84enZ2WVZRUXllMHFWTTFEYlkwa09GZjBmK0xFbklxNgpUQXFTZi9waWlKcUpIM25lWUlqR21heVJCNGtPUS9OQmVuWnorUFVkKzU2RXJ0Z0dScTMwWW90eXBEejRxKzA5CkhtbzdXdEFIMmhVNWhLVGEzUUc0bkdBbEpVRmQ3ZFFxQzFTTzhrZWNMWENZdFUrWlY3NjBWb2ZnVDBlRE1GVFEKUmg0R0Jla0NnWUVBNUNLSXl1b1VyUXg3REZMaVBCTitKTm90bzgza2k3ZUxpclBsanY0bGwvd3pvZ0xYUVRpTwptZDF1dmk5U2d2U2RQcEVCWTZVRmxhamlLSS9oN011UkMyY3FCdjdqazZWazkvb2FsVjBWVXRCZk9LQ29wb2doCktWSDFCVTc2SU43RHlrQUpHQ0NhaGNFbldtekFzdXdlQWZJZkRFKzF0a2hHOVhJVFYweWgwNjBDZ1lFQXpXN20KaGZaVHFQdjMxKzdQcExFZmNFbHE0Y2l0TjdBRk1LY2hKb0dCRmFGWGc0T2RJb2hWWFZOb3ZkTDVWUkhMbDNwUgpkZW9IUTRxSzZqV0tPV01qcy9wS2NER0k2N3R4NEhQdDRlMFlGZ1BTMFVZbWpZeElkMGJ3SzYzcng5SXV4M1hFCmVReEREY3lFWXRCbUFoUTNZMTN6ditWRmtGMzZHM1dpc1hVbTJ0OENnWUJxYm80aEZLb0d2YzdlUmdEVUJFZ1MKaTFORm0zWG5sUDdkKytXNkcybVFpWkhSSU1BcDVtZm84cnlLcitzdnUwMXM5aHVPMEZ0Vm9nKzQycitOU0w5bgpjWDdTK3JGVG5aTUllYjlUTmJVUUNMU1Q1NmdtNFZXUFFIUXVRTlZDNW9xelhjS2daZjJSTHpiYjRlYlkwbjJCCmJPTDlUR2E3SHVjejlUOSt0L3E3bFFLQmdDaS8vdXBEMm9TQ3RyOFdtQW5MT0xsRlZ2WkNvRm1UaVBRRnN3VzQKV3FxM3ZteFFCek42WjdTRGZ4dG9aaDBCMHFqUmtxY1pMU2V3cTYyWndUbHcrUHdTZ2dHUFVlR3c1UDNwQVI3MwpzUGRzK3J5WWRiMU9QbkdxbUttUmJsdk16WXF6U2EzWlNOUEw1ZGJVRCtnSnFwTURaLzZBdERQVzhHM1IvOXZECnFWbHhBb0dCQU5WMFFFeTM0ZFNLbjdqSVV4NzdLM0k4S3hraFFmS2pzQXRiS3dsQmtkVFNJVjhER3RvZUlXcmUKK2hXWVA1RXk1RWJnYXhuRFpPblN2d05LSy80VGU4TmJPdHJuRVBRUjhGMUdrWVkyZjAyU1pmTDNiY1RZOThFeQo5ZERpbG1RbmFEOWp4R2hFTVducUNUUVRVeWZRVFlUdk1TTjNxRExPYkNLaHJoclR1a0N1Ci0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg==
        sshPublicKey: c3NoLXJzYSBBQUFBQjNOemFDMXljMkVBQUFBREFRQUJBQUFCQVFDM0VudlByUDVBYkJWVnB4REVjUm9WSWJnL1VyV2xiSkNhVkNmei9EaittR0pVbjA3b0FLdXhOQ2JXTTF6a2JXTFh4V1ZGa3laeDZTNEhuK1VDK1JqL1k2VXFYclk5MS9weWhSZUkzNGl2WmNrUkZZdlRHcE9raXZ6ZFdMT0tjajZsWjh2SGF0MnVON2R3N284UWsvd09TNDJzRzBBUk83d0JPbG5GcEZ3VWkrWE41NXNoK210eENtREoxNTVqdGd1QWtnUENiRHpvYXdCZEs2bnRNSUU3bklpOVJ4ODlSVENwZk1lU3VsUnpVOGNvMmRQMVBtNndYWDRDYTkydzE1K3d3b2F1SFJKNTB2eGppdXNwTENhTngwU2hiWlRJdWFFUVZkMVdyTXEzaEQ0U3lBWTBlZUIzZXVueFE5Z2ZVS20zbHJlLzdtRTVyaGk2V3E0TDVyV3o=
        vmSize: Standard_B2ms
    versions:
      kubelet: v1.13.5
```


#### Create new machinesets

You can create new machinesets by deploying the following yaml


```yaml
kind: MachineSet
apiVersion: cluster.k8s.io/v1alpha1
metadata:
  name: machineset-1
spec:
  replicas: 1
  selector:
    matchLabels:
      cluster.k8s.io/cluster-name: az1
      cluster.pharmer.io/mg: Standard_B2ms
  template:
    metadata:
      creationTimestamp: '2019-05-16T08:56:21Z'
      labels:
        cluster.k8s.io/cluster-name: az1
        cluster.pharmer.io/cluster: az1
        cluster.pharmer.io/mg: Standard_B2ms
        node-role.kubernetes.io/node: ''
        set: node
    spec:
      metadata:
        creationTimestamp: 
      providerSpec:
        value:
          kind: AzureMachineProviderSpec
          apiVersion: azureprovider/v1alpha1
          metadata:
            creationTimestamp: 
          roles:
          - Node
          location: eastus2
          vmSize: Standard_B2ms
          image:
            publisher: Canonical
            offer: UbuntuServer
            sku: 16.04-LTS
            version: latest
          osDisk:
            osType: Linux
            managedDisk:
              storageAccountType: Premium_LRS
            diskSizeGB: 30
          sshPublicKey: c3NoLXJzYSBBQUFBQjNOemFDMXljMkVBQUFBREFRQUJBQUFCQVFDM0VudlByUDVBYkJWVnB4REVjUm9WSWJnL1VyV2xiSkNhVkNmei9EaittR0pVbjA3b0FLdXhOQ2JXTTF6a2JXTFh4V1ZGa3laeDZTNEhuK1VDK1JqL1k2VXFYclk5MS9weWhSZUkzNGl2WmNrUkZZdlRHcE9raXZ6ZFdMT0tjajZsWjh2SGF0MnVON2R3N284UWsvd09TNDJzRzBBUk83d0JPbG5GcEZ3VWkrWE41NXNoK210eENtREoxNTVqdGd1QWtnUENiRHpvYXdCZEs2bnRNSUU3bklpOVJ4ODlSVENwZk1lU3VsUnpVOGNvMmRQMVBtNndYWDRDYTkydzE1K3d3b2F1SFJKNTB2eGppdXNwTENhTngwU2hiWlRJdWFFUVZkMVdyTXEzaEQ0U3lBWTBlZUIzZXVueFE5Z2ZVS20zbHJlLzdtRTVyaGk2V3E0TDVyV3o=
          sshPrivateKey: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb3dJQkFBS0NBUUVBdHhKN3o2eitRR3dWVmFjUXhIRWFGU0c0UDFLMXBXeVFtbFFuOC93NC9waGlWSjlPCjZBQ3JzVFFtMWpOYzVHMWkxOFZsUlpNbWNla3VCNS9sQXZrWS8yT2xLbDYyUGRmNmNvVVhpTitJcjJYSkVSV0wKMHhxVHBJcjgzVml6aW5JK3BXZkx4MnJkcmplM2NPNlBFSlA4RGt1TnJCdEFFVHU4QVRwWnhhUmNGSXZsemVlYgpJZnByY1FwZ3lkZWVZN1lMZ0pJRHdtdzg2R3NBWFN1cDdUQ0JPNXlJdlVjZlBVVXdxWHpIa3JwVWMxUEhLTm5UCjlUNXVzRjErQW12ZHNOZWZzTUtHcmgwU2VkTDhZNHJyS1N3bWpjZEVvVzJVeUxtaEVGWGRWcXpLdDRRK0VzZ0cKTkhuZ2QzcnA4VVBZSDFDcHQ1YTN2KzVoT2E0WXVscXVDK2Exc3dJREFRQUJBb0lCQUJOSjNrT21UVytLTThFLwpoZk84bXV2cERwbVZaRkFXblRHMWRqUXR1ZStSTEtNUDJlZDEwcUVzQm4rQkQrTjladkdtK2FHWC9HLzZDb0NCCkowYmw2ZTFXbVZ0YWVVY1F6M0ZyZG14VWFQbFo5eEpXdTlHMU5pTWJCY05vaWhvbktWU1NHQlZkdkJlVUJUN2YKMDdFQ2RvY25ETGs2Y2NpZkM1THhpKzNZQUYrbHBCVk84enZ2WVZRUXllMHFWTTFEYlkwa09GZjBmK0xFbklxNgpUQXFTZi9waWlKcUpIM25lWUlqR21heVJCNGtPUS9OQmVuWnorUFVkKzU2RXJ0Z0dScTMwWW90eXBEejRxKzA5CkhtbzdXdEFIMmhVNWhLVGEzUUc0bkdBbEpVRmQ3ZFFxQzFTTzhrZWNMWENZdFUrWlY3NjBWb2ZnVDBlRE1GVFEKUmg0R0Jla0NnWUVBNUNLSXl1b1VyUXg3REZMaVBCTitKTm90bzgza2k3ZUxpclBsanY0bGwvd3pvZ0xYUVRpTwptZDF1dmk5U2d2U2RQcEVCWTZVRmxhamlLSS9oN011UkMyY3FCdjdqazZWazkvb2FsVjBWVXRCZk9LQ29wb2doCktWSDFCVTc2SU43RHlrQUpHQ0NhaGNFbldtekFzdXdlQWZJZkRFKzF0a2hHOVhJVFYweWgwNjBDZ1lFQXpXN20KaGZaVHFQdjMxKzdQcExFZmNFbHE0Y2l0TjdBRk1LY2hKb0dCRmFGWGc0T2RJb2hWWFZOb3ZkTDVWUkhMbDNwUgpkZW9IUTRxSzZqV0tPV01qcy9wS2NER0k2N3R4NEhQdDRlMFlGZ1BTMFVZbWpZeElkMGJ3SzYzcng5SXV4M1hFCmVReEREY3lFWXRCbUFoUTNZMTN6ditWRmtGMzZHM1dpc1hVbTJ0OENnWUJxYm80aEZLb0d2YzdlUmdEVUJFZ1MKaTFORm0zWG5sUDdkKytXNkcybVFpWkhSSU1BcDVtZm84cnlLcitzdnUwMXM5aHVPMEZ0Vm9nKzQycitOU0w5bgpjWDdTK3JGVG5aTUllYjlUTmJVUUNMU1Q1NmdtNFZXUFFIUXVRTlZDNW9xelhjS2daZjJSTHpiYjRlYlkwbjJCCmJPTDlUR2E3SHVjejlUOSt0L3E3bFFLQmdDaS8vdXBEMm9TQ3RyOFdtQW5MT0xsRlZ2WkNvRm1UaVBRRnN3VzQKV3FxM3ZteFFCek42WjdTRGZ4dG9aaDBCMHFqUmtxY1pMU2V3cTYyWndUbHcrUHdTZ2dHUFVlR3c1UDNwQVI3MwpzUGRzK3J5WWRiMU9QbkdxbUttUmJsdk16WXF6U2EzWlNOUEw1ZGJVRCtnSnFwTURaLzZBdERQVzhHM1IvOXZECnFWbHhBb0dCQU5WMFFFeTM0ZFNLbjdqSVV4NzdLM0k4S3hraFFmS2pzQXRiS3dsQmtkVFNJVjhER3RvZUlXcmUKK2hXWVA1RXk1RWJnYXhuRFpPblN2d05LSy80VGU4TmJPdHJuRVBRUjhGMUdrWVkyZjAyU1pmTDNiY1RZOThFeQo5ZERpbG1RbmFEOWp4R2hFTVducUNUUVRVeWZRVFlUdk1TTjNxRExPYkNLaHJoclR1a0N1Ci0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg==
      versions:
        kubelet: v1.13.5
```


#### Create new machine-deployments

You can create new machine-deployments by deploying the following yaml

 
```yaml
kind: MachineDeployment
apiVersion: cluster.k8s.io/v1alpha1
metadata:
  name: machinedeployment-1
spec:
  replicas: 1
  selector:
    matchLabels:
      cluster.k8s.io/cluster-name: az1
      cluster.pharmer.io/mg: Standard_B2ms
  template:
    metadata:
      creationTimestamp: '2019-05-16T08:56:21Z'
      labels:
        cluster.k8s.io/cluster-name: az1
        cluster.pharmer.io/cluster: az1
        cluster.pharmer.io/mg: Standard_B2ms
        node-role.kubernetes.io/node: ''
        set: node
    spec:
      metadata:
        creationTimestamp: 
      providerSpec:
        value:
          kind: AzureMachineProviderSpec
          apiVersion: azureprovider/v1alpha1
          metadata:
            creationTimestamp: 
          roles:
          - Node
          location: eastus2
          vmSize: Standard_B2ms
          image:
            publisher: Canonical
            offer: UbuntuServer
            sku: 16.04-LTS
            version: latest
          osDisk:
            osType: Linux
            managedDisk:
              storageAccountType: Premium_LRS
            diskSizeGB: 30
          sshPublicKey: c3NoLXJzYSBBQUFBQjNOemFDMXljMkVBQUFBREFRQUJBQUFCQVFDM0VudlByUDVBYkJWVnB4REVjUm9WSWJnL1VyV2xiSkNhVkNmei9EaittR0pVbjA3b0FLdXhOQ2JXTTF6a2JXTFh4V1ZGa3laeDZTNEhuK1VDK1JqL1k2VXFYclk5MS9weWhSZUkzNGl2WmNrUkZZdlRHcE9raXZ6ZFdMT0tjajZsWjh2SGF0MnVON2R3N284UWsvd09TNDJzRzBBUk83d0JPbG5GcEZ3VWkrWE41NXNoK210eENtREoxNTVqdGd1QWtnUENiRHpvYXdCZEs2bnRNSUU3bklpOVJ4ODlSVENwZk1lU3VsUnpVOGNvMmRQMVBtNndYWDRDYTkydzE1K3d3b2F1SFJKNTB2eGppdXNwTENhTngwU2hiWlRJdWFFUVZkMVdyTXEzaEQ0U3lBWTBlZUIzZXVueFE5Z2ZVS20zbHJlLzdtRTVyaGk2V3E0TDVyV3o=
          sshPrivateKey: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb3dJQkFBS0NBUUVBdHhKN3o2eitRR3dWVmFjUXhIRWFGU0c0UDFLMXBXeVFtbFFuOC93NC9waGlWSjlPCjZBQ3JzVFFtMWpOYzVHMWkxOFZsUlpNbWNla3VCNS9sQXZrWS8yT2xLbDYyUGRmNmNvVVhpTitJcjJYSkVSV0wKMHhxVHBJcjgzVml6aW5JK3BXZkx4MnJkcmplM2NPNlBFSlA4RGt1TnJCdEFFVHU4QVRwWnhhUmNGSXZsemVlYgpJZnByY1FwZ3lkZWVZN1lMZ0pJRHdtdzg2R3NBWFN1cDdUQ0JPNXlJdlVjZlBVVXdxWHpIa3JwVWMxUEhLTm5UCjlUNXVzRjErQW12ZHNOZWZzTUtHcmgwU2VkTDhZNHJyS1N3bWpjZEVvVzJVeUxtaEVGWGRWcXpLdDRRK0VzZ0cKTkhuZ2QzcnA4VVBZSDFDcHQ1YTN2KzVoT2E0WXVscXVDK2Exc3dJREFRQUJBb0lCQUJOSjNrT21UVytLTThFLwpoZk84bXV2cERwbVZaRkFXblRHMWRqUXR1ZStSTEtNUDJlZDEwcUVzQm4rQkQrTjladkdtK2FHWC9HLzZDb0NCCkowYmw2ZTFXbVZ0YWVVY1F6M0ZyZG14VWFQbFo5eEpXdTlHMU5pTWJCY05vaWhvbktWU1NHQlZkdkJlVUJUN2YKMDdFQ2RvY25ETGs2Y2NpZkM1THhpKzNZQUYrbHBCVk84enZ2WVZRUXllMHFWTTFEYlkwa09GZjBmK0xFbklxNgpUQXFTZi9waWlKcUpIM25lWUlqR21heVJCNGtPUS9OQmVuWnorUFVkKzU2RXJ0Z0dScTMwWW90eXBEejRxKzA5CkhtbzdXdEFIMmhVNWhLVGEzUUc0bkdBbEpVRmQ3ZFFxQzFTTzhrZWNMWENZdFUrWlY3NjBWb2ZnVDBlRE1GVFEKUmg0R0Jla0NnWUVBNUNLSXl1b1VyUXg3REZMaVBCTitKTm90bzgza2k3ZUxpclBsanY0bGwvd3pvZ0xYUVRpTwptZDF1dmk5U2d2U2RQcEVCWTZVRmxhamlLSS9oN011UkMyY3FCdjdqazZWazkvb2FsVjBWVXRCZk9LQ29wb2doCktWSDFCVTc2SU43RHlrQUpHQ0NhaGNFbldtekFzdXdlQWZJZkRFKzF0a2hHOVhJVFYweWgwNjBDZ1lFQXpXN20KaGZaVHFQdjMxKzdQcExFZmNFbHE0Y2l0TjdBRk1LY2hKb0dCRmFGWGc0T2RJb2hWWFZOb3ZkTDVWUkhMbDNwUgpkZW9IUTRxSzZqV0tPV01qcy9wS2NER0k2N3R4NEhQdDRlMFlGZ1BTMFVZbWpZeElkMGJ3SzYzcng5SXV4M1hFCmVReEREY3lFWXRCbUFoUTNZMTN6ditWRmtGMzZHM1dpc1hVbTJ0OENnWUJxYm80aEZLb0d2YzdlUmdEVUJFZ1MKaTFORm0zWG5sUDdkKytXNkcybVFpWkhSSU1BcDVtZm84cnlLcitzdnUwMXM5aHVPMEZ0Vm9nKzQycitOU0w5bgpjWDdTK3JGVG5aTUllYjlUTmJVUUNMU1Q1NmdtNFZXUFFIUXVRTlZDNW9xelhjS2daZjJSTHpiYjRlYlkwbjJCCmJPTDlUR2E3SHVjejlUOSt0L3E3bFFLQmdDaS8vdXBEMm9TQ3RyOFdtQW5MT0xsRlZ2WkNvRm1UaVBRRnN3VzQKV3FxM3ZteFFCek42WjdTRGZ4dG9aaDBCMHFqUmtxY1pMU2V3cTYyWndUbHcrUHdTZ2dHUFVlR3c1UDNwQVI3MwpzUGRzK3J5WWRiMU9QbkdxbUttUmJsdk16WXF6U2EzWlNOUEw1ZGJVRCtnSnFwTURaLzZBdERQVzhHM1IvOXZECnFWbHhBb0dCQU5WMFFFeTM0ZFNLbjdqSVV4NzdLM0k4S3hraFFmS2pzQXRiS3dsQmtkVFNJVjhER3RvZUlXcmUKK2hXWVA1RXk1RWJnYXhuRFpPblN2d05LSy80VGU4TmJPdHJuRVBRUjhGMUdrWVkyZjAyU1pmTDNiY1RZOThFeQo5ZERpbG1RbmFEOWp4R2hFTVducUNUUVRVeWZRVFlUdk1TTjNxRExPYkNLaHJoclR1a0N1Ci0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg==
      versions:
        kubelet: v1.13.5
```


#### Scale Cluster

You can also update number of nodes of an existing machine-set and machine-deployment using

```console
$ kubectl edit <machineset-name> 
$ kubectl edit <machinedeployment-name> 
```
and update the `spec.replicas` field

#### Delete nodes

You can delete machines using

```console
$ kubectl delete machine <machine-name>
```
Warning: if the machine is controlled by a machineset, a new machine will be created. You should update/delete machineset in that case

You can delete machine-set and machine-deployments using

```console
$ kubectl delete machineset <machineset-name>
$ kubectl delete machinedeployment <machinedeployment-name>
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
....
....
status:
  phase: Deleting
...
...
```


Here,

- `metadata.deletionTimestamp`: is set when cluster deletion command was applied.

Now, to apply delete operation of the cluster, run

```console
$ pharmer apply az1
```

**Congratulations !!!** , you're an official `pharmer` user now.

