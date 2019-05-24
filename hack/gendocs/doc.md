---
title: {{ .Provider.Capital }} Overview
menu:
product_pharmer_{{ .Release }}
identifier: {{ .Provider.Small }}-overview
name: Overview
parent: {{ .Provider.Small }}
weight: 10
product_name: pharmer
menu_name: product_pharmer_{{ .Release }}
section_menu_id: cloud
url: /products/pharmer/{{ .Release }}/cloud/{{ .Provider.Small}}/
aliases:
- /products/pharmer/{{ .Release }}/cloud/{{ .Provider.Small }}/README/
---

# Running Kubernetes on [{{ .Provider.Capital}}]({{ .Provider.URL }})

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

{{ template "credential-importing" . }}

### Cluster provisioning

There are two steps to create a Kubernetes cluster using `pharmer`. In first step `pharmer` create basic configuration file with user choice. Then in second step `pharmer` applies those information to create cluster on specific provider.

Here, we discuss how to use `pharmer` to create a Kubernetes cluster on `{{ .Provider.Small }}`

#### Cluster Creating

We want to create a cluster with following information:

- Provider: {{ .Provider.Small }}
- Cluster name: {{ .Provider.Small }}
- Location: {{ .Provider.Location }}
- Number of master nodes: {{ .Provider.MasterNodeCount }}
- Number of worker nodes: 1
- Worker Node sku: {{ .Provider.NodeSpec.SKU }} (cpu: {{ .Provider.NodeSpec.CPU }}, memory: {{ .Provider.NodeSpec.Memory }})
- Kubernetes version: {{ .KubernetesVersion }}
- Credential name: [{{ .Provider.Small }}](#credential-importing)

For location code and sku details click [hrere](https://github.com/pharmer/cloud/blob/master/data/json/apis/cloud.pharmer.io/v1/cloudproviders/{{ .Provider.Small }}.json)

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
$ pharmer create cluster {{ .Provider.Small }}-1 \
    --masters {{ .Provider.MasterNodeCount }} \
    --provider {{ .Provider.Small }} \
    --zone {{ .Provider.Location }} \
    --nodes {{.Provider.NodeSpec.SKU }}=1 \
    --credential-uid {{.Provider.Small }} \
    --kubernetes-version {{ .KubernetesVersion }}
```

To know about [pod networks](https://kubernetes.io/docs/concepts/cluster-administration/networking/) supports in `pharmer` click [here](/docs/networking.md)

The directory structure of the storage provider will be look like:

{{ template "tree" . }}

Here,
  - `machine`: conntains information about the master machines to be deployed
  - `machineset`: contains information about the machinesets to be deployed
  - `pki`: contains the cluster certificate information containing `ca`, `front-proxy-ca`, `etcd/ca` and service account keys `sa`
  - `ssh`: has the ssh credentials on cluster's nodes. With this key you can `ssh` into any node on a cluster
  - `{{ .Provider.ClusterName }}.json`: contains the cluster resource information

You can view your cluster configuration file by following command.

{{ template "pending-cluster" . }}

You can modify this configuration by:
```console
$ pharmer edit cluster {{ .Provider.ClusterName }}
```

#### Applying 

If everything looks ok, we can now apply the resources. This actually creates resources on `{{ .Provider.Small }}`.
Up to now we've only been working locally.

To apply run:

 ```console
$ pharmer apply {{ .Provider.ClusterName }}
```

Now, `pharmer` will apply that configuration, this create a Kubernetes cluster. After completing task the configuration file of the cluster will be look like

{{ template "ready-cluster" . }}

Here,

  - `status.phase`: is ready. So, you can use your cluster from local machine.
  - `status.clusterApi.status.apiEndpoints` is the cluster's apiserver address

To get the `kubectl` configuration file(kubeconfig) on your local filesystem run the following command.

```console
$ pharmer use cluster {{ .Provider.ClusterName }}
```
If you don't have `kubectl` installed click [here](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

Now you can run `kubectl get nodes` and verify that your kubernetes {{ .KubernetesVersion }} is running.

{{ template "get-nodes" . }}

{{ template "ssh" . }}

### Cluster Scaling

Scaling a cluster refers following meanings
- Add new master and worker machines
- Increment the number of nodes of a certain machine-set and machine-deployment
- Decrement the number of nodes of a certain machine-set and machine-deployment
- Introduce a new machine-set and machine-deployment with a number of nodes
- Delete existing machine, machine-set and machine-deployments

You can see the machine and machine-sets deployed in the cluster

{{ template "get-machines" . }}

{{ if .Provider.HASupport }}
#### Deploy new master machines
You can create new master machine by the deploying the following yaml
{{ template "master-machine" . }}
{{ end }} 

#### Create new worker machines

You can create new worker machines by deploying the following yaml

{{ template "worker-machine" . }}

#### Create new machinesets

You can create new machinesets by deploying the following yaml

{{ template "machineset" . }}

#### Create new machine-deployments

You can create new machine-deployments by deploying the following yaml

{{ template "machinedeployment" . }}

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
$ pharmer delete cluster {{ .Provider.ClusterName }}
```

Then, the yaml file looks like

{{ template "deleted-cluster" . }}

Here,

- `metadata.deletionTimestamp`: is set when cluster deletion command was applied.

Now, to apply delete operation of the cluster, run

```console
$ pharmer apply {{ .Provider.ClusterName }}
```

**Congratulations !!!** , you're an official `pharmer` user now.

