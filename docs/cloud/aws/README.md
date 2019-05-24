---
title: AWS Overview
menu:
product_pharmer_0.3.1
identifier: aws-overview
name: Overview
parent: aws
weight: 10
product_name: pharmer
menu_name: product_pharmer_0.3.1
section_menu_id: cloud
url: /products/pharmer/0.3.1/cloud/aws/
aliases:
- /products/pharmer/0.3.1/cloud/aws/README/
---

# Running Kubernetes on [AWS](https://aws.amazon.com)

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


#### Setup IAM User

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
apiVersion: v1beta1
kind: Credential
metadata:
  creationTimestamp: "2019-04-04T09:33:32Z"
  name: aws
spec:
  data:
    accessKeyID: <key-id>
    secretAccessKey: <access-key>
  provider: aws
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
aws          aws            accessKeyID=AKIAJKUZAD3HM7OEKPNA, secretAccessKey=*****
```

You can also see the stored credential from the following location:
```console
~/.pharmer/store.d/$USER/credentials/aws.json
```

#### Cluster IAM User

 While creating cluster within AWS `pharmer` creates following IAM roles and policies
 * [IAM master policy](https://github.com/pharmer/pharmer/blob/0.3.1/cloud/providers/aws/iam.go#L6)
 * [IAM controller policy](https://github.com/pharmer/pharmer/blob/0.3.1/cloud/providers/aws/iam.go#L77)
 * [IAM master role](https://github.com/pharmer/pharmer/blob/0.3.1/cloud/providers/aws/iam.go#L160)
 * [IAM node policy](https://github.com/pharmer/pharmer/blob/0.3.1/cloud/providers/aws/iam.go#L175)
 * [IAM node role](https://github.com/pharmer/pharmer/blob/0.3.1/cloud/providers/aws/iam.go#L200)


### Cluster provisioning

There are two steps to create a Kubernetes cluster using `pharmer`. In first step `pharmer` create basic configuration file with user choice. Then in second step `pharmer` applies those information to create cluster on specific provider.

Here, we discuss how to use `pharmer` to create a Kubernetes cluster on `aws`

#### Cluster Creating

We want to create a cluster with following information:

- Provider: aws
- Cluster name: aws
- Location: us-east-1b
- Number of master nodes: 3
- Number of worker nodes: 1
- Worker Node sku: t2.medium (cpu: 2, memory: 4 Gb)
- Kubernetes version: v1.13.5
- Credential name: [aws](#credential-importing)

For location code and sku details click [hrere](https://github.com/pharmer/cloud/blob/master/data/json/apis/cloud.pharmer.io/v1/cloudproviders/aws.json)

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
$ pharmer create cluster aws-1 \
    --masters 3 \
    --provider aws \
    --zone us-east-1b \
    --nodes t2.medium=1 \
    --credential-uid aws \
    --kubernetes-version v1.13.5
```

To know about [pod networks](https://kubernetes.io/docs/concepts/cluster-administration/networking/) supports in `pharmer` click [here](/docs/networking.md)

The directory structure of the storage provider will be look like:


```console
$ tree ~/.pharmer/store.d/$USER/clusters/
/home/<user>/.pharmer/store.d/<user>/clusters/
├── aws-1
│   ├── machine
│   │   ├── aws-1-master-0.json
│   │   ├── aws-1-master-1.json
│   │   └── aws-1-master-2.json
│   ├── machineset
│   │   └── t2.medium-pool.json
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
│       ├── id_aws-1-sshkey
│       └── id_aws-1-sshkey.pub
└── aws-1.json

6 directories, 15 files
```


Here,
  - `machine`: conntains information about the master machines to be deployed
  - `machineset`: contains information about the machinesets to be deployed
  - `pki`: contains the cluster certificate information containing `ca`, `front-proxy-ca`, `etcd/ca` and service account keys `sa`
  - `ssh`: has the ssh credentials on cluster's nodes. With this key you can `ssh` into any node on a cluster
  - `aws-1.json`: contains the cluster resource information

You can view your cluster configuration file by following command.

 
```yaml
$ pharmer get cluster aws-1 -o yaml
apiVersion: cluster.pharmer.io/v1beta1
kind: Cluster
metadata:
  creationTimestamp: "2019-05-16T06:26:31Z"
  generation: 1557987991689291249
  name: aws-1
  uid: 8b9832d9-77a3-11e9-a68a-e0d55ee85d14
spec:
  clusterApi:
    apiVersion: cluster.k8s.io/v1alpha1
    kind: Cluster
    metadata:
      creationTimestamp: null
      name: aws-1
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
          apiVersion: awsprovider/v1alpha1
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
          kind: AWSClusterProviderSpec
          metadata:
            creationTimestamp: null
            name: aws-1
          networkSpec:
            vpc: {}
          region: us-east-1
          saKeyPair:
            cert: null
            key: null
          sshKeyName: aws-1-sshkey
    status: {}
  config:
    apiServerExtraArgs:
      cloud-provider: aws
      kubelet-preferred-address-types: InternalIP,InternalDNS,ExternalDNS,ExternalIP
    cloud:
      aws:
        bastionSGName: aws-1-bastion
        iamProfileMaster: master.aws-1.pharmer
        iamProfileNode: node.aws-1.pharmer
        masterIPSuffix: ".9"
        masterSGName: aws-1-controlplane
        nodeSGName: aws-1-node
        privateSubnetCidr: 10.0.0.0/24
        publicSubnetCidr: 10.0.1.0/24
        vpcCIDR: 10.0.0.0/16
        vpcCIDRBase: "10.0"
      cloudProvider: aws
      networkProvider: calico
      region: us-east-1
      sshKeyName: aws-1-sshkey
      zone: us-east-1b
    credentialName: aws
    kubernetesVersion: v1.13.5
    masterCount: 3
status:
  cloud:
    aws: {}
    loadBalancer:
      dns: ""
      ip: ""
      port: 0
  phase: Pending
```


You can modify this configuration by:
```console
$ pharmer edit cluster aws-1
```

#### Applying 

If everything looks ok, we can now apply the resources. This actually creates resources on `aws`.
Up to now we've only been working locally.

To apply run:

 ```console
$ pharmer apply aws-1
```

Now, `pharmer` will apply that configuration, this create a Kubernetes cluster. After completing task the configuration file of the cluster will be look like

 
```yaml
$ pharmer get cluster aws-1 -o yaml
apiVersion: cluster.pharmer.io/v1beta1
kind: Cluster
metadata:
  creationTimestamp: "2019-05-16T06:40:11Z"
  generation: 1557988811187811647
  name: aws-1
  uid: 740dbfc4-77a5-11e9-9087-e0d55ee85d14
spec:
  clusterApi:
    apiVersion: cluster.k8s.io/v1alpha1
    kind: Cluster
    metadata:
      creationTimestamp: null
      name: aws-1
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
          apiVersion: awsprovider/v1alpha1
          caKeyPair:
            cert: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUN1RENDQWFDZ0F3SUJBZ0lCQURBTkJna3Foa2lHOXcwQkFRc0ZBREFOTVFzd0NRWURWUVFERXdKallUQWUKRncweE9UQTFNVFl3TmpRd01URmFGdzB5T1RBMU1UTXdOalF3TVRGYU1BMHhDekFKQmdOVkJBTVRBbU5oTUlJQgpJakFOQmdrcWhraUc5dzBCQVFFRkFBT0NBUThBTUlJQkNnS0NBUUVBcndqQ2dKQS9oVnJ4UmZKMUVyU0o3ZTBFCjZVcjBiQnB0eWczN0dYaTFXbUk3TEtKMUlTSEFTMlV1eW91anZVWUYxRnZXbStYU3c4dzVMU1g5OFAzNy92UnUKYnQ5dHN0MnpFSlUyMldZUnNFNldSSVJRelZGQ21ZVGNvbXdMVzNEUW13WWJpV1NCZnFIbmo3TlNla2xBbGNOZQp1d0llVGs0UGIxSUlvNlNMYVhQMjZRbHlqNnZBRHBteVZFZmRwaUpVRVlWZU9VejRubjlPOFpNbzlLWnVWTHJmCm85a2NVVGZKWDBuUHBIODYyUVlCeDZuTDBkNEZFOUhHOW90YTNKakFGcFVTYTV6bmc4OE9iKzJPQWJTVXV1aGoKc0xTWjREcEIybXpCbDhtOVdlSnFRMUhsUHFxUFgzS0FoNG05R0tybFdKRndyS0hoaktzdTJ1cUpNWVZ4blFJRApBUUFCb3lNd0lUQU9CZ05WSFE4QkFmOEVCQU1DQXFRd0R3WURWUjBUQVFIL0JBVXdBd0VCL3pBTkJna3Foa2lHCjl3MEJBUXNGQUFPQ0FRRUFMVnZ5QWJnQkxxRWdJZ1R4bjVwd0RHUVRQSm1BWjNrSzJ5RVQwb21RVFpnK3B0dVcKNEFRdjIxQmw0YjBUcEw3Q091Ym9rZEJGcUcvSXlPMWpoV3RxTXlhNHAzUnVmbUQzbW1hYkVsQTJNUzcyOHJUTwpyR3RzNFVQU1RGYzJUdnhYMXJxMUpQK3orOXBMU1lVa3RQTUNneVMxZ2lhR2dJcWRzdjZHY0w1ZUNLTjZQRXNqCjdyTEtBQ1c1NGd6WjlaRndiZk8vSmUzOUZiT3lOcDErNjV6cUhJRXZQQmh2SVJ3QUdTVUJDZlNEaGxiYWdESEQKb251NDM5MkNzeXdFTjRKNnpweTRmbzRFeW9JOTFERUNGa3A4anoya1BXUlFzd2d2V3hJZHM4c1JBWWVrSlpmZQpyOXdGcWlYMEFvQU5IR3ZKL3EvSlhvREU2bllyYWFscnhZVC9HQT09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
            key: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb2dJQkFBS0NBUUVBcndqQ2dKQS9oVnJ4UmZKMUVyU0o3ZTBFNlVyMGJCcHR5ZzM3R1hpMVdtSTdMS0oxCklTSEFTMlV1eW91anZVWUYxRnZXbStYU3c4dzVMU1g5OFAzNy92UnVidDl0c3QyekVKVTIyV1lSc0U2V1JJUlEKelZGQ21ZVGNvbXdMVzNEUW13WWJpV1NCZnFIbmo3TlNla2xBbGNOZXV3SWVUazRQYjFJSW82U0xhWFAyNlFseQpqNnZBRHBteVZFZmRwaUpVRVlWZU9VejRubjlPOFpNbzlLWnVWTHJmbzlrY1VUZkpYMG5QcEg4NjJRWUJ4Nm5MCjBkNEZFOUhHOW90YTNKakFGcFVTYTV6bmc4OE9iKzJPQWJTVXV1aGpzTFNaNERwQjJtekJsOG05V2VKcVExSGwKUHFxUFgzS0FoNG05R0tybFdKRndyS0hoaktzdTJ1cUpNWVZ4blFJREFRQUJBb0lCQUJwYVBFWjA4VXRYbk5uRwpIa0E1dEVhQkYrc3o0TWJoMThQREJSb1pwVnc5UytGMWVDTUwzTks5SWlWV2pzbHhZSEZQZm1rc1dlWW11amtFCjdrMjVQNVpzSUxCS3JVNXZ1SVQzb2pGcS82REd4REwrcE5lMHMwMC94cVFobGpnbkxSRWFBMDFWTjNYa1ZHTzcKUU9DdVpLM25venlPbmhkMkF6YmtaKzZUV0hZQXdzN0tCeHorbnlab2NLMkxLNVprV1NSc2NNbWNBTndaT2xDVAozbUNybzV0ZGhJTGc2c3NkemlHL3J3VEV1NVZWVDAxYzdncm9CSy9mclYxOWJIVFloVGNBRVNUWHJ4dk03eFo0CjlYQUQ1b3BYdERBUWI3WHdMTUhPNXd3YXA1aEpHTVdPSU9HejRtNWpTTnhwU0dzRUxtczhrMlBKTWpOelBvMEEKZ25KV0IxVUNnWUVBeWwvNTlCYnNGTGtXZnEwQ3h1bHllY3YvUTZub1h6YzlFQnlRYkpDYlMyUzJsejFLZTUvQgpON3RXRkt5amJRWjM1RjZHd1l4Q2hSazdTc1VjU2swaFF3UVFOSy95cGVMckE3c0xFM09lZlhBSXFTZEVqcGNrCnArOFhScWdNNzFGVVJZVmVESUFxNnNoM3BuWmxPaWNYall5Ly9SdXZyekRjWFBvSXVDczNjRzhDZ1lFQTNXb2oKWlh4cDhXa3dnem96WkJDUnFpWStzb3VJVnVDblNtOC8xbk9jTUN0Z05ac3BvYzdYNklyRG9seE1aWDdpazFicQoyOGhYMGQxQ3JTV3BCd0Z4ck9BbS9qNmRRbGVsa2FtTHBKWXl4WlhNQmNpWkJHWjdyVk9kc0dpNkkrbkVOaWtCCkpTZ2dobHRRU0ZEQmxsNW1Tb0JldTJZUjB2ZkFqTTVvV3lKZGJMTUNnWUF1eXF3dmJORmZKVUIzUDZlQnVGNVkKblB0RGVOaWFrMW9TREppMEVXZG1zajJwa0dsVlZpWEZyaElFSzdxSnJkSXd3azVrRi9zVmJUVVJYNnZmM2grUApzRFBUQ3MrTzNYMjdXaGNBZzE0azRLK1A5TjFjSHNSQjgvMHN3QlJsalNkdi81czBScm9sbVA1WlJjeTMrbXZ1CnRabDZlMWxPcDN4OEh1Ky9MWGJmRHdLQmdGRngyR0ZkV1c4V3ZXU1lCUTFhMXVvYXRWZGg0aDNxOXo1M0c2bGIKejJrY200QThlaHp1QkJlaTY0R09wLzl4cEJDRW1WR05LVmltSmYrZzZjTU04ZTZnYVZkK1dzUnJqeGk4b0FSRAp4NXRNbGNiTzJoSjNUQ2tlcDlPYlFsVXhpUjVQQ1AvTStlSFNOdjdTemRMdEdIMXhLT2VRRFNCb1IrakRpRGwxCnM1M25Bb0dBSU9aZEtPbHVqVTdzelBNdmlHZkREbnpGL3RWU3NjeGRrR05nVHRvYVJGcllZTnpseGpEcGJkM3QKdXpCSTJ5OWpjbXZqcW1VVld1NFo2YlBTeTRoeWJlRXNrNXBCd1Z6WGg1NFdMaE9yRzBCRzJrT1dka0lCN0ZJMApWYnpYUkRvVWVQdHBaQ1QwNHZvOCt0dHp4eGFyRk9CaERlN09qbmJOdERaNm1Vc3lGcXc9Ci0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg==
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
            cert: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUN5RENDQWJDZ0F3SUJBZ0lCQURBTkJna3Foa2lHOXcwQkFRc0ZBREFWTVJNd0VRWURWUVFERXdwcmRXSmwKY201bGRHVnpNQjRYRFRFNU1EVXhOakEyTkRBeE1sb1hEVEk1TURVeE16QTJOREF4TWxvd0ZURVRNQkVHQTFVRQpBeE1LYTNWaVpYSnVaWFJsY3pDQ0FTSXdEUVlKS29aSWh2Y05BUUVCQlFBRGdnRVBBRENDQVFvQ2dnRUJBT0RqCmxpeFU3Ty9OQ3NlZ3lKZ2pnUVBvaG9FTjgxUm41bEd0UXB0eE1BWEUvSTBKU0JQUHBLUHpNU0tQZUdKdklFc04KdE1sWGRSR3VVbDBMa09BY2NUN1FQbGl5ZFhpbkVac3JYd0QyRWdoc09ZV0VnZHlhTE9Wd3ZvOUpiS1d6WkpsNwo3eEoyYy8rR3hSQWFBZ3htelpXY1pZc1phUEd4N1FDSFFkWXZzQ0pvZHU5d1hBU3RpQyt2QjMvU3o1ejUxbXJSCkxuM3krVGFKSlBTWVdCQ1EzYytQbWJQdlZyVnl0TC9iWjFEMm1paWQ5YkVzNkVPWmZwTGF6N1c4V05BeUZQUXoKWTdGUEpobkNweExnUTVIRjk4bFNDblhuL3kvbjFwanVjZGR2bzVYQzNmOU1WWHZaaVpTdG5IR21UM3pBemRMSgpOQjVWSlBEWmh1QWlGSHI0dzVjQ0F3RUFBYU1qTUNFd0RnWURWUjBQQVFIL0JBUURBZ0trTUE4R0ExVWRFd0VCCi93UUZNQU1CQWY4d0RRWUpLb1pJaHZjTkFRRUxCUUFEZ2dFQkFCdzVHVzVQOTRQWkF6bThrRm80VTEwZlNFSFcKN1VQdzUxTzhhLzFVeHJsdVRVa0kxcFl3K2RVSFR0WmloZHMxNjNsTklPSlV1eDl3V2FYSGJVT1hkOTNDSWVYMgpHMTFYSUtmNHFSVmFUYmJnNlFrOXFQUmQ1azMzb0NQYmRrOFdJYXdWZzBTdjlNeHRJV0Z0eUNla2Z6UTNUaXU0CjRKR2ZPa2p2MXRSZUlNYXdxaXVwVVpWSk5Tby80M28wWHFrQnZKQTNOSkVtMGRMSVZUQmUwdVluOHRVd0RDRGYKOFZ0ZTMyZ3FqdGhTMGI1ZWNLMnJvWVNBNzB6ZW1MODZwYmx6eWxsK0hScFpRcTRqbm4yckMzUk9vclA3bE9VTgpDaU1uVDBHNnVTU0ZVTHR4TUtvc2s5NE1qZ2FhQXRwOUlWUDJsSjZ2akE4blpJYUZsYUthNGZuQkMwWT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=
            key: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb3dJQkFBS0NBUUVBNE9PV0xGVHM3ODBLeDZESW1DT0JBK2lHZ1EzelZHZm1VYTFDbTNFd0JjVDhqUWxJCkU4K2tvL014SW85NFltOGdTdzIweVZkMUVhNVNYUXVRNEJ4eFB0QStXTEoxZUtjUm15dGZBUFlTQ0d3NWhZU0IKM0pvczVYQytqMGxzcGJOa21YdnZFblp6LzRiRkVCb0NER2JObFp4bGl4bG84Ykh0QUlkQjFpK3dJbWgyNzNCYwpCSzJJTDY4SGY5TFBuUG5XYXRFdWZmTDVOb2trOUpoWUVKRGR6NCtacys5V3RYSzB2OXRuVVBhYUtKMzFzU3pvClE1bCtrdHJQdGJ4WTBESVU5RE5qc1U4bUdjS25FdUJEa2NYM3lWSUtkZWYvTCtmV21PNXgxMitqbGNMZC8weFYKZTltSmxLMmNjYVpQZk1ETjBzazBIbFVrOE5tRzRDSVVldmpEbHdJREFRQUJBb0lCQUJGRlNSNGtjNEhEQkdYcQpVaDFrOUo2Qk4vc25RQjJtVVFqS3ZvZkRmSVdrNkNSSXB6Rm1TK1dQWXFHZDFRZnlNcyt3d01hSm9lTDJ1VHFPCkRkVTRPZll4OWVmSDVMK2NUUHpXcXRnZkRhbDU3anp6dlNsYzZiL0JGZEZaT1MvTWhCaEpiVVhFdFFuVnBzS2kKeks5NUlhYXd1UmVpbnUyWTFYT3A3NE5zYkNGb01ZUXViYzVYcVFjbHdSU2hOekRMUmZ6Tlh4RVM0SzNTNTN2TwpjRnRoUy9YdDY5YzJzZ2wydElSMTBncWJsSUpibWYrcjByZ0p1US80YjNpZDR4cUs3bjFPMndBUmJJWm9CQXVZCmF5OUN5c0k5NnBEWkUzYlpKWkVvU3p3aC9TS0ViRmFWNUlYQjZpbGs1S3RKNERHRVNRM2hCNGdhbkx6RlhPaGgKU1ZRSi9xa0NnWUVBOW05bktEZ256TzJYMUZva2NKNUJ4U0JrR2dCQjlYdU1VSEkxOWhoTTZEMnZxUGNNNEdsdAphQkNoZjRIVzMwZ3UzSi9DeThtZWFZam1wNkZucFV5K1RMZDhzampoU05NN1ROTlhJczN3U294bFdGZDdoK2F6ClRlOWhzTjRVa2UrSDA5UFJydVM4VVRFU1kxR21jNkREbWdRS3ZHcURUSU5qR3BEaVBJMnl2NlVDZ1lFQTZaNFoKZXNRb0xGYmF0ZVpYQzlUVmgzSFQ0aW5ldEFxUGdkR1ZmalZzQmVMQlIrbVJLdytCVzhoakIzbksrKzl1N1Z5WgoybHdtM3dsZWNNTUlFQjJSdlZjTE5zb2xnVkpOUGhiU0ZCMjJpL2lVV3JIeXlGNFVVTXcvRUdjQi9mQnB5cGJLCmdJL0VkMWZubmp2bGI0eWR6WUxacUxxVHpJNkU3NDF5VU5DUjBZc0NnWUVBcTYyQVYreUhEYVNYbVVBVEVzR3QKWC83b3ZaUmdYdnZyREVBRWg2VDJMdlNLWTFONGpQM2xVaElEOENncjRQRVFkSEozNmpCVFE0SXo3YVByNktSRwpEbnZsU3VPRlRvNlpTVFFTQ0JVZnlVOTFhczNIS1MzMnk5eHdXaDdjaGE0eEdjaiswckJXNE5rbXpqb2JrNEh6CndsLytlclJaTS94MEZoWEpCaFpRNkdFQ2dZQUt6OGRNR0RIbncybEJ5OVF4ZHZzZTFwVEF2Y0ZSd2I5Y2VhK2EKZEt4NEpVTmVxWitQUjV0ci9QMGdSbkd2Y1NoSUVlWUk1Z1RpZzVOOFVucFlESlpIRmZVdDV2TVBaaGl2QysxVApBd0VFdjA5V1Z5L3VOL1Jtdk4wYVREb2FYM3IxNWo2ZTdvaGdJWkJWa1Y0UDZJa1JEc0kxL2RTRFBnRkcrTnZXCmc2Q0wrd0tCZ0hEckgxcXlvd0h6L1VxazNmdWV3Y0ZMRkpXZ3JtTXR5RWg2RHRRdWpuL2wzTzZMcjd1K0N2ZXQKMnVzeVR3TWdrOGs4ZXM5ZDZCbU53N3RkQTZxSE5mYnVtZUZONlZFSFQ4dDd6NlhTWUdhRlhtWE5pWUtoRC9FSApLMnBVMHg4VWY3bURiYzlhc0VUYlg5ZmRXUUl0dDNMR3pkMEVnQksxTE9ETGdYMDZWSk55Ci0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg==
          frontProxyCAKeyPair:
            cert: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUN1RENDQWFDZ0F3SUJBZ0lCQURBTkJna3Foa2lHOXcwQkFRc0ZBREFOTVFzd0NRWURWUVFERXdKallUQWUKRncweE9UQTFNVFl3TmpRd01USmFGdzB5T1RBMU1UTXdOalF3TVRKYU1BMHhDekFKQmdOVkJBTVRBbU5oTUlJQgpJakFOQmdrcWhraUc5dzBCQVFFRkFBT0NBUThBTUlJQkNnS0NBUUVBNWNZWkk0REY2a3JkVzJkLzFjWjVqc1RvCkpwQnJFdXBsUTNCWlF5UGp6Q1U0MmE3UjBFZkNLR3B3Vkt6am0rdDJJdXh3QjZFbGQxbCsxQktQNHJJa25ra1EKZElCSWg3cjFnQ0FDQWtQMVpvSzRkZUt1NkNYM2E1K21RSFMrb0xQelp2NytXSUs4b3NRVXlGdVJwVGxkbEQrVgp6bC82aWQ3dXNrWkRmNFNYbmk4OUIwZzB0bmwwdXRYOTRlQ2tTR1BhNWRvRytVM0p2VVBIb295aXQyUTFta3loCitQY3Y1TWlmRXRKZHNaMXJ2MktSRWx3Z0k2SVlTRkxHQVdMNjU2VnhSSjZ6OUpIUFFMVEtMMTR4VUxlSEpjMnEKWEpYbG4xdlJ3UTRlUFkyajVrM0dneU9tU0t2L2dQVHZ4Z0hWL2ZET1ppeFd1TWU3RTM1R1A4Z0RIcVhuRndJRApBUUFCb3lNd0lUQU9CZ05WSFE4QkFmOEVCQU1DQXFRd0R3WURWUjBUQVFIL0JBVXdBd0VCL3pBTkJna3Foa2lHCjl3MEJBUXNGQUFPQ0FRRUFlVmVneHl3cmdqL0ZZbFM1R2gzUEFDcG82bGpiRWpKMU1JMDhuMEFRVXprYXErQVIKcmJZV2tmbml5WW9zOHhpWXdacEtWTUxHT1pqNkJ2WFdnRkUvUlNreTBRQXhUYi9MalhFUGJqdzdFNThIZng3dQpRZ3VBTmpIOCtQNFdzQzMxQ041SFVFSHdaNk5nS25ORlhaczd3dEVZL2tqZ1U3NUhzUEtHSTBBNm9SZndzSDduCmxYQ0hoNmQyZVBoZlVneFRjOENoTVNSTHBqY2JxMkczeGY5VFhyR3ZYUEZPUThWbktpYTFuaTNUUlpldVBKOFAKd1JvNkNleWd6bVVTSXlyQ0s2SnNOTkxRa0Y3Tmp5WTNNTEN5UytwRC9GZmtMcHJycXRCUTIzeGVtdDl6M1Y5ZQpDL0RjeEVPWDFzMUdhNkoxTHJ2SWdZY0FkTENFZVNRNmwzOFk5UT09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
            key: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFcEFJQkFBS0NBUUVBNWNZWkk0REY2a3JkVzJkLzFjWjVqc1RvSnBCckV1cGxRM0JaUXlQanpDVTQyYTdSCjBFZkNLR3B3Vkt6am0rdDJJdXh3QjZFbGQxbCsxQktQNHJJa25ra1FkSUJJaDdyMWdDQUNBa1AxWm9LNGRlS3UKNkNYM2E1K21RSFMrb0xQelp2NytXSUs4b3NRVXlGdVJwVGxkbEQrVnpsLzZpZDd1c2taRGY0U1huaTg5QjBnMAp0bmwwdXRYOTRlQ2tTR1BhNWRvRytVM0p2VVBIb295aXQyUTFta3loK1BjdjVNaWZFdEpkc1oxcnYyS1JFbHdnCkk2SVlTRkxHQVdMNjU2VnhSSjZ6OUpIUFFMVEtMMTR4VUxlSEpjMnFYSlhsbjF2UndRNGVQWTJqNWszR2d5T20KU0t2L2dQVHZ4Z0hWL2ZET1ppeFd1TWU3RTM1R1A4Z0RIcVhuRndJREFRQUJBb0lCQUN3NHJQdmxPN0gweUpkZgoydjJFbmo1NDdRa0hBR1I2a2hTaG1ieFBPdmMrTHF2T2RuajBab3lxdDRYRVpHWE1za2JVWkZkRGoxZGg5UVBSCnNybDVlWXl4R1NhaXpkSzVpNmdtQU56NHdWRUNWWWZ5b2FEeU5hVVQ2OFk1OGJveUIrVkpyQi9Td1lVOTRaWVIKOFh6d0JtK1NzVDB2d2FNcE1aMUQ0cGUzS0FyOFJSSFpMYkZJeWVpdjFibGtFWE8rTkxWcmxGckt0K0ErMmQ2YwoyNjkzQ1g4bkZxRnVlSy84UHArTXhVdnA0VTYvQWJpazk3ZmlQT0VLQzJlMmZuRWIxQXFpKzNtMjZ2L1BBcUhECm5GTFdTUWpWc0NuVG1SdWFWS0dyZlpmYVpXM1lxTkxFaGVXRWducEJJQWFhNlY0NFdFcU9ieDZaZ0JvbTlUNUEKa1pXaVlOa0NnWUVBOUlETGFTS2p1emZYdFVjMyt4Um5oNmdMaXpsQ2xHUWFjRnhrMUZLbzArSkFxZDNiVFpSMgpkMzlyS2VXRzFLSTJBbDRmbG8yQm0vcmJ6bHhUUlRzd1dHdlJiZ3lqRWVERzNqbjVycE1aMVYrYnpNbHdVVlNiCnZiVjBFMElOc3NRSkxUUTc4OWtwK0p5Sk56cHN1ejljWm52RFRqdGcrZUtraTRucThEMzB5VDBDZ1lFQThKUC8KK3g5eDRkVExJTWNjeDlCbHh0WnphdUJRVXlUeHIzNWNzV0laV3ZvRTFFV2d5SUh5Z1VwamVJeEZ2QStycFE2Kwp0a01ZUC83NnlNUEZwUDk4S0Q0VXZCNTdUSWdZazNobTE5Q0tVRmsxUm45U2FtVlpveE1yZTMvVlV0dFBQYTVhClFlUDdQT0xEdUl6NzhvMys1REY1Vm4wYXluUDllOWtLajhJUXJ1TUNnWUVBc0xCaVRwZTV1cEdnVUdBbkZFcXEKaGwzcCtiSm5hdFRzUmtaK2x2RWxEL2x3d1ZDU0tuNGZIanYyTlZDcEh3QWFCNXY5Tjg4SzJxMXVLcktOZW5wTApkWnAwdmhKanhZZXFMdTIyZ1hITU9XWGVNUjloQzJVWkp2NzU0dkRZOVZhMVN2VjBYY09Sa1JlT0VWc25PQ21SCm5IM1RwYlZEWDFGcGwyMFRXb2xyWEFFQ2dZQmJZUXZoR2QrSzFPWFc3R3B0SnlZUmNaRmpiaEowa2xyT3V1T0EKYU8rU2s5YlR2aUxGSmo2emgwcmpGZnpDNHZ6aWRBaFNlSWUwZnloSXE3dmQ0VUVLbEJJU0prM1ZFdmlxd3hmbApMNHZwMndpV1gzUXhmNCtkbG9GMHBtaWowVE4zRFV1eExXUlhpeGFtZWI2Vk1nUTRMdWFVeUc0dHFnTUZVTHBuClFtSk4yd0tCZ1FEenVmTnFYdE5MYTFOZHJFVzhCL2FHaVhzOGFhUUcyci81TmxOeE00OW51Z09XTy9TS0pXVW8KQ2QzdEt3Y09icGVwMkpZN0tqTVpyUWxieWtiWmZiOTNVb1F4WnoycjNMUi91SkRlVVpqSUZhQ0IrUkVSSEd6ZAp5TWkyU1NzV1daNHpadUNObytQVkl2bG5zMEc3V3FNWXl4TkVxV0pxODlDT3poT0dwU2RrRUE9PQotLS0tLUVORCBSU0EgUFJJVkFURSBLRVktLS0tLQo=
          kind: AWSClusterProviderSpec
          metadata:
            creationTimestamp: null
            name: aws-1
          networkSpec:
            subnets:
            - availabilityZone: us-east-1b
              cidrBlock: 10.0.1.0/24
              id: subnet-085c030daeb5f069c
              isPublic: true
              natGatewayId: nat-0c86a686879d47e47
              routeTableId: rtb-07b4766903951a56f
            - availabilityZone: us-east-1b
              cidrBlock: 10.0.0.0/24
              id: subnet-0e6ceedb9f4965e54
              isPublic: false
              routeTableId: rtb-06aae6fd3689a9865
            vpc:
              cidrBlock: 10.0.0.0/16
              id: vpc-030839a2833a7f522
              internetGatewayId: igw-026492e984a415ca9
              tags:
                sigs.k8s.io/cluster-api-provider-aws/managed: "true"
          region: us-east-1
          saKeyPair:
            cert: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUM1RENDQWN5Z0F3SUJBZ0lCQURBTkJna3Foa2lHOXcwQkFRc0ZBREFqTVNFd0h3WURWUVFERXhoellTMWoKWlhKMGFXWnBZMkYwWlMxaGRYUm9iM0pwZEhrd0hoY05NVGt3TlRFMk1EWTBNREV5V2hjTk1qa3dOVEV6TURZMApNREV5V2pBak1TRXdId1lEVlFRREV4aHpZUzFqWlhKMGFXWnBZMkYwWlMxaGRYUm9iM0pwZEhrd2dnRWlNQTBHCkNTcUdTSWIzRFFFQkFRVUFBNElCRHdBd2dnRUtBb0lCQVFERURBOWh3WGl1RU5Ra2lNTHhaUm5VSEIvV2NoWDAKRkkzVDduejBTaE1KM09zU3ZUZE5vdkdEMjdQdTg3cWRhSUlTWjZDQmMybERoYmw4MjZtL3c1ZXZ4bU91K3BTUApaUTVTQUYrSlZkbXNNK0ZNTnZZU05CRndIT0t3MXFvUmRBa05peUtIOTlVcmxtelkyN1lDdE1WOXZMN0hldllrCkhoRzg0VDZpQTZwNGNXK0dZOFpxNVloOThSUnBndU9Zb1RmSFlGZDk5a2ZYWk9MNzc2dDJ3Nk1QSEthM3JSbWEKWjZWK0xpTENhUE81UThvS0hoN1JubDRXYVVFbndvcHpZRExOY2tNTlhvYTBuc0hXcitXZzByMUhBTFhISk9vVgo2cktWd1hvTFRsK3dua0s1eTByeGRFUmRrRnk5alpLVUFaVHlmUHZvY2RlRS91SVNFWS9sTTlnUkFnTUJBQUdqCkl6QWhNQTRHQTFVZER3RUIvd1FFQXdJQ3BEQVBCZ05WSFJNQkFmOEVCVEFEQVFIL01BMEdDU3FHU0liM0RRRUIKQ3dVQUE0SUJBUUEyc2Fkd1MzbVltSnBvYi9Fd3NSb3F3VjRlYVBzUUN0aHZVdXgwT3drcHFKYWgweWJ4eWthUQpCUVlIcllrLzJ5SSt4ZXQwc29IYUVKTGZrMkZjZmdsWjM5WFJMZnBTblJTMENZajdJeWFqMTdGT1QzUXN2b0kwCjVUdUw3bngzQWhVZ1I4K2dnR0tXSVRuWnlEODNTa3h5bERjSU5DSFNpZ04vcmpuVHVHWXN0Qit5Q3BZeStjVXkKUGpiR0VTdnBvM1BYR2FKSnlKQTNtaWxjWFdmbUFoaG8zbGFuOGFOeE5STlpKSkhQOElJbkVYdHZ0NUNsT0tDcgo2dGhsYXF5TWJjRnoydCtxdWVtKy93UjQ4bkJ3VEJDSm5iaG9FWHQrNmk5aHloL3hoZ3RlSVYxejlkeCs1Z1FHCnJGVFo3aU8wQVZMWW1XWjh0MityRFQySGZzNi95clBkCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
            key: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb3dJQkFBS0NBUUVBeEF3UFljRjRyaERVSklqQzhXVVoxQndmMW5JVjlCU04wKzU4OUVvVENkenJFcjAzClRhTHhnOXV6N3ZPNm5XaUNFbWVnZ1hOcFE0VzVmTnVwdjhPWHI4WmpydnFVajJVT1VnQmZpVlhackRQaFREYjIKRWpRUmNCemlzTmFxRVhRSkRZc2loL2ZWSzVaczJOdTJBclRGZmJ5K3gzcjJKQjRSdk9FK29nT3FlSEZ2aG1QRwphdVdJZmZFVWFZTGptS0UzeDJCWGZmWkgxMlRpKysrcmRzT2pEeHltdDYwWm1tZWxmaTRpd21qenVVUEtDaDRlCjBaNWVGbWxCSjhLS2MyQXl6WEpERFY2R3RKN0IxcS9sb05LOVJ3QzF4eVRxRmVxeWxjRjZDMDVmc0o1Q3VjdEsKOFhSRVhaQmN2WTJTbEFHVThuejc2SEhYaFA3aUVoR1A1VFBZRVFJREFRQUJBb0lCQUUvMC96MEdkRnJCNEZQNgpOMC9PeFNiK1JYbm4wODVWcDdhZEdQZGxVcmgrRXAzMDhCNUk2Nm0wcklFemhKUDRjTHhpNlZLQ3FKYnlia0ZmCk1hOVZiWU15TGF2SzVWWktoL21uejA4cTVYbFhPM2NqSDE4elB6MXplbjFYUDh1WWdLeTJaMkgvRVVFU3U5Z0MKWEF4a2YvdVZSRllGYjJneG4xaGlvWEhnZnVGWjQ3UkgxV0JqampuWjVpL1RMdURTRWNMMTlwU0J6NDl1RE9FUQp1UUQzLzNZbmNYRFlpU2RsRWl2NEIxZFg0czZoVnYyTzRiQXgzdWg5N3BTbDVIV293L1J4QTZFYldDVlZYWEJLCmRNZTBVVVBhOVJwSzl3RTVvTEhlQlI1Y3JWY1BqeWx3M2gvMS9iWGlUOHVlSXMvVmhpSDUrYVZOODI1T29TczAKM2NjMnhNVUNnWUVBNUtpMDU4enNUanFQWlRmU0VrOHhsUUg3RDlhbzV1MWNNaGpwWXNLeWhZeGsrZy9jQzFUYgppY2UvcURBZ0JpcUFXODg1dWFJWlZncXQzZlFHWGlKTHgzREQydklqYmNiNFZvbTBLVW0zRndyai9zSzIwbDY4Cko0aElaVkVCL2o1VDZiZU8zSUdVdnZtbUpqbjBxY1hiTmI2WnZFTlpQSWlwUkRBYmVoZ3A5eDhDZ1lFQTIzMFkKNkJqSFFWczRGQzdmRDZNWHRUQkQ2QVE5V3NYTHhqNW9pa0ZweWs3MkFta2tGWVhJUFp0L1ZSbXBpVEJkYzZsYQpISUFacnorUlN3K2FrSElEMkIrWWRyT1ZDYzd3ZWJWd0V3UWdhMWNseTMzcGJpYlU0TnFOTWEvUUZkTDZOYTFLCnZDUk4vZ3pkSDVoeXk4UW9QOW14dkdqbm1Xd01FTkRKekdXNE9zOENnWUVBdmhCSmh4LzhFQzUzQVJCOEtrSHYKbWNjeXRBQ2ZGb3lZQlFCV0JvU0Z0YUowVUxNY0djTW9WUWRYRk9zancxeFNvMzNGb3JyTnlvcEg2V1VzWWRTcQpIcFpxQmpVZEkrT3VpdWdkZS9CTkl2Y25lcHpKTUdZVWlkdXJLYVJEUHR6NkRSeEp3SnBwVkxEWTNZOXhBaWwzClE5NHhsWjU1cjJwOUlEUElzeDBnek1zQ2dZQmdEOGsxMDVwcGpVM203M2lxOUZ0czdublo4dmtUWUZ4R0lJeEsKYmtTcHlaTThETjVCR1ROQlcyd0lSOW4rZEErQ2pvMGt5aC96cG1PbHNXZVpibjBtT3ZYVWhkWmwyNDgrQlYzTwp4TkNYaWlXOWdSY0lJYkNyMUp0Vk1yaGt4TmpEWTF2QktqYUVTUWNDVEF0NkNSa0FrUHVNRlhHL29SMUt3c1ovClVjbW0yd0tCZ0Y4U3RuRllOWFFyNWc2R1hYQyt5WlVLUWpwVG84VmdDZktkcnFxUDNTaXgzckt6bTJmM1MzckQKM09UcUl6YXgrY3EwZzFRdkU4YmtZZCtuamM3NFgxR3dkQ2pEelh1NFhFUGhDQnhDMk5JcHQ0b0lPRGdZVU1nQwo2L1ZQTHpMSk1lM0JtcjVRU0lDb0I4N1RpK1hWcm44TkNVZzR2Sjd5YkQrMy9JWGVwQ0JlCi0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg==
          sshKeyName: aws-1-sshkey
    status:
      apiEndpoints:
      - host: aws-1-apiserver-1328099475.us-east-1.elb.amazonaws.com
        port: 6443
      providerStatus:
        bastion:
          ebsOptimized: false
          enaSupport: true
          id: i-0d7016536acfd17b9
          imageId: ami-41e0b93b
          instanceState: running
          keyName: aws-1-sshkey
          privateIp: 10.0.1.245
          publicIp: 34.205.72.251
          securityGroupIds:
          - sg-0f4d0cf3df30c6efd
          subnetId: subnet-085c030daeb5f069c
          tags:
            Name: aws-1-bastion
            kubernetes.io/cluster/aws-1: owned
            sigs.k8s.io/cluster-api-provider-aws/managed: "true"
            sigs.k8s.io/cluster-api-provider-aws/role: bastion
          type: t2.micro
        metadata:
          creationTimestamp: null
        network:
          apiServerElb:
            attributes:
              idleTimeout: 600000000000
            dnsName: aws-1-apiserver-1328099475.us-east-1.elb.amazonaws.com
            name: aws-1-apiserver
            scheme: internet-facing
            securityGroupIds:
            - sg-0f10cf6bc1e94a354
            subnetIds:
            - subnet-085c030daeb5f069c
          securityGroups:
            bastion:
              id: sg-0f4d0cf3df30c6efd
              ingressRule:
              - cidrBlocks:
                - 0.0.0.0/0
                description: SSH
                fromPort: 22
                protocol: tcp
                sourceSecurityGroupIds: null
                toPort: 22
              name: aws-1-bastion
              tags:
                Name: aws-1-bastion
                kubernetes.io/cluster/aws-1: owned
                sigs.k8s.io/cluster-api-provider-aws/managed: "true"
                sigs.k8s.io/cluster-api-provider-aws/role: bastion
            controlplane:
              id: sg-0f10cf6bc1e94a354
              ingressRule:
              - cidrBlocks:
                - 0.0.0.0/0
                description: Kubernetes API
                fromPort: 6443
                protocol: tcp
                sourceSecurityGroupIds: null
                toPort: 6443
              - cidrBlocks: null
                description: SSH
                fromPort: 22
                protocol: tcp
                sourceSecurityGroupIds:
                - sg-0f4d0cf3df30c6efd
                toPort: 22
              - cidrBlocks: null
                description: etcd
                fromPort: 2379
                protocol: tcp
                sourceSecurityGroupIds:
                - sg-0f10cf6bc1e94a354
                toPort: 2379
              - cidrBlocks: null
                description: etcd peer
                fromPort: 2380
                protocol: tcp
                sourceSecurityGroupIds:
                - sg-0f10cf6bc1e94a354
                toPort: 2380
              - cidrBlocks: null
                description: bgp (calico)
                fromPort: 179
                protocol: tcp
                sourceSecurityGroupIds:
                - sg-0b732196e2c4771ee
                - sg-0f10cf6bc1e94a354
                toPort: 179
              - cidrBlocks: null
                description: IP-in-IP (calico)
                fromPort: 0
                protocol: "4"
                sourceSecurityGroupIds:
                - sg-0b732196e2c4771ee
                - sg-0f10cf6bc1e94a354
                toPort: 0
              name: aws-1-controlplane
              tags:
                Name: aws-1-controlplane
                kubernetes.io/cluster/aws-1: owned
                sigs.k8s.io/cluster-api-provider-aws/managed: "true"
                sigs.k8s.io/cluster-api-provider-aws/role: controlplane
            node:
              id: sg-0b732196e2c4771ee
              ingressRule:
              - cidrBlocks:
                - 0.0.0.0/0
                description: Node Port Services
                fromPort: 30000
                protocol: tcp
                sourceSecurityGroupIds: null
                toPort: 32767
              - cidrBlocks: null
                description: SSH
                fromPort: 22
                protocol: tcp
                sourceSecurityGroupIds:
                - sg-0f4d0cf3df30c6efd
                toPort: 22
              - cidrBlocks: null
                description: Kubelet API
                fromPort: 10250
                protocol: tcp
                sourceSecurityGroupIds:
                - sg-0f10cf6bc1e94a354
                toPort: 10250
              - cidrBlocks: null
                description: bgp (calico)
                fromPort: 179
                protocol: tcp
                sourceSecurityGroupIds:
                - sg-0b732196e2c4771ee
                - sg-0f10cf6bc1e94a354
                toPort: 179
              - cidrBlocks: null
                description: IP-in-IP (calico)
                fromPort: 0
                protocol: "4"
                sourceSecurityGroupIds:
                - sg-0b732196e2c4771ee
                - sg-0f10cf6bc1e94a354
                toPort: 0
              name: aws-1-node
              tags:
                Name: aws-1-node
                kubernetes.io/cluster/aws-1: owned
                sigs.k8s.io/cluster-api-provider-aws/managed: "true"
                sigs.k8s.io/cluster-api-provider-aws/role: node
  config:
    apiServerExtraArgs:
      cloud-provider: aws
      kubelet-preferred-address-types: InternalIP,InternalDNS,ExternalDNS,ExternalIP
    caCertName: ca
    cloud:
      aws:
        bastionSGName: aws-1-bastion
        iamProfileMaster: master.aws-1.pharmer
        iamProfileNode: node.aws-1.pharmer
        masterIPSuffix: ".9"
        masterSGName: aws-1-controlplane
        nodeSGName: aws-1-node
        privateSubnetCidr: 10.0.0.0/24
        publicSubnetCidr: 10.0.1.0/24
        vpcCIDR: 10.0.0.0/16
        vpcCIDRBase: "10.0"
      cloudProvider: aws
      instanceImage: ami-d15a75c7
      networkProvider: calico
      os: ubuntu
      region: us-east-1
      sshKeyName: aws-1-sshkey
      zone: us-east-1b
    credentialName: aws
    frontProxyCACertName: front-proxy-ca
    kubernetesVersion: v1.13.5
    masterCount: 3
status:
  cloud:
    aws:
      bastionSGID: sg-0f4d0cf3df30c6efd
      masterSGID: sg-0f10cf6bc1e94a354
      nodeSGID: sg-0b732196e2c4771ee
    loadBalancer:
      dns: aws-1-apiserver-1328099475.us-east-1.elb.amazonaws.com
      ip: ""
      port: 6443
  phase: Ready
```


Here,

  - `status.phase`: is ready. So, you can use your cluster from local machine.
  - `status.clusterApi.status.apiEndpoints` is the cluster's apiserver address

To get the `kubectl` configuration file(kubeconfig) on your local filesystem run the following command.

```console
$ pharmer use cluster aws-1
```
If you don't have `kubectl` installed click [here](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

Now you can run `kubectl get nodes` and verify that your kubernetes v1.13.5 is running.


NAME                         STATUS   ROLES    AGE     VERSION
ip-10-0-0-101.ec2.internal   Ready    master   5m37s   v1.13.5
ip-10-0-0-239.ec2.internal   Ready    master   6m28s   v1.13.5
ip-10-0-0-27.ec2.internal    Ready    node     5m37s   v1.13.5
ip-10-0-0-71.ec2.internal    Ready    master   8m36s   v1.13.5




You can ssh to the nodes from bastion node.

First, ssh to bastion node
```console
$ cd ~/.pharmer/store.d/$USER/clusters/aws-1/ssh/
$ ssh-add id_aws-1-sshkey
Identity added: id_aws-1-sshkey (id_aws-1-sshkey)
$ ssh -A ubuntu@34.205.72.251 #bastion-ip
```
Then you can ssh to any node in the cluster from bastion node using its private ip

```console
ubuntu@ip-10-0-1-245:~$ ssh 10.0.0.71
ubuntu@ip-10-0-0-71:~$ 
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
NAME                   AGE
aws-1-master-0         27m
aws-1-master-1         27m
aws-1-master-2         27m
t2.medium-pool-4mnwg   27m

$ kubectl get machinesets
NAME             AGE
t2.medium-pool   27m
```



#### Deploy new master machines
You can create new master machine by the deploying the following yaml

```yaml
apiVersion: cluster.k8s.io/v1alpha1
kind: Machine
metadata:
  labels:
    cluster.k8s.io/cluster-name: aws-1
    node-role.kubernetes.io/master: ""
    set: controlplane
  name: new-master
  namespace: default
spec:
  providerSpec:
    value:
      apiVersion: awsprovider/v1alpha1
      kind: AWSMachineProviderSpec
      iamInstanceProfile: master.aws-1.pharmer
      instanceType: t2.large
      keyName: aws-1-sshkey
  versions:
    controlPlane: v1.13.5
    kubelet: v1.13.5

 

#### Create new worker machines

You can create new worker machines by deploying the following yaml


```yaml
apiVersion: cluster.k8s.io/v1alpha1
kind: Machine
metadata:
  labels:
    cluster.k8s.io/cluster-name: aws-1
    node-role.kubernetes.io/node: ""
    set: node
  name: new-node
  namespace: default
spec:
  providerSpec:
    value:
      apiVersion: awsprovider/v1alpha1
      kind: AWSMachineProviderSpec
      iamInstanceProfile: node.aws-1.pharmer
      instanceType: t2.large
      keyName: aws-1-sshkey
  versions:
    kubelet: v1.13.5
```


#### Create new machinesets

You can create new machinesets by deploying the following yaml


```yaml
apiVersion: cluster.k8s.io/v1alpha1
kind: MachineSet
metadata:
  name: new-machineset
  namespace: default
spec:
  replicas: 2
  selector:
    matchLabels:
      cluster.k8s.io/cluster-name: aws-1
      cluster.pharmer.io/mg: t2.medium
  template:
    metadata:
      labels:
        cluster.k8s.io/cluster-name: aws-1
        cluster.pharmer.io/cluster: aws-1
        cluster.pharmer.io/mg: t2.medium
        node-role.kubernetes.io/node: ""
        set: node
    spec:
      providerSpec:
        value:
          apiVersion: awsprovider/v1alpha1
          iamInstanceProfile: node.aws-1.pharmer
          instanceType: t2.medium
          keyName: aws-1-sshkey
          kind: AWSMachineProviderSpec
      versions:
        kubelet: v1.13.5
```


#### Create new machine-deployments

You can create new machine-deployments by deploying the following yaml

 

```yaml
apiVersion: cluster.k8s.io/v1alpha1
kind: MachineDeployment
metadata:
  name: new-machinedeployment
  namespace: default
spec:
  replicas: 2
  selector:
    matchLabels:
      cluster.k8s.io/cluster-name: aws-1
      cluster.pharmer.io/mg: t2.medium
  template:
    metadata:
      labels:
        cluster.k8s.io/cluster-name: aws-1
        cluster.pharmer.io/cluster: aws-1
        cluster.pharmer.io/mg: t2.medium
        node-role.kubernetes.io/node: ""
        set: node
    spec:
      providerSpec:
        value:
          apiVersion: awsprovider/v1alpha1
          iamInstanceProfile: node.aws-1.pharmer
          instanceType: t2.medium
          keyName: aws-1-sshkey
          kind: AWSMachineProviderSpec
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
$ pharmer delete cluster aws-1
```

Then, the yaml file looks like


```yaml
$ pharmer get cluster a1 -o yaml
kind: Cluster
apiVersion: cluster.pharmer.io/v1beta1
metadata:
  name: aws-1
  uid: 740dbfc4-77a5-11e9-9087-e0d55ee85d14
  generation: 1557988811187811600
  creationTimestamp: '2019-05-16T06:40:11Z'
  deletionTimestamp: '2019-05-16T08:18:45Z'
...
...
...
status:
  phase: Deleting
...
...
```


Here,

- `metadata.deletionTimestamp`: is set when cluster deletion command was applied.

Now, to apply delete operation of the cluster, run

```console
$ pharmer apply aws-1
```

**Congratulations !!!** , you're an official `pharmer` user now.

