---
title: Pharmer Create Cluster
menu:
  product_pharmer_0.1.0-alpha.1:
    identifier: pharmer-create-cluster
    name: Pharmer Create Cluster
    parent: reference
product_name: pharmer
left_menu: product_pharmer_0.1.0-alpha.1
section_menu_id: reference
---
## pharmer create cluster

Create a Kubernetes cluster for a given cloud provider

### Synopsis


Create a Kubernetes cluster for a given cloud provider

```
pharmer create cluster [flags]
```

### Examples

```
pharmer create cluster demo-cluster
```

### Options

```
      --credential-uid string       Use preconfigured cloud credential uid
  -h, --help                        help for cluster
      --kubeadm-version string      Kubeadm version
      --kubelet-version string      kubelet/kubectl version
      --kubernetes-version string   Kubernetes version
      --network-provider string     Name of CNI plugin. Available options: calico, flannel, kubenet, weavenet (default "calico")
      --nodes stringToInt           Node set configuration (default [])
      --provider string             Provider name
      --zone string                 Cloud provider zone name
```

### Options inherited from parent commands

```
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

### SEE ALSO
* [pharmer create](/docs/reference/pharmer_create.md)	 - 

