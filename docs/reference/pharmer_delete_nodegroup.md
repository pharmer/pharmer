---
title: Pharmer Delete Nodegroup
menu:
  product_pharmer_0.1.0-alpha.1:
    identifier: pharmer-delete-nodegroup
    name: Pharmer Delete Nodegroup
    parent: reference
product_name: pharmer
menu_name: product_pharmer_0.1.0-alpha.1
section_menu_id: reference
---
## pharmer delete nodegroup

Delete a Kubernetes cluster NodeGroup

### Synopsis


Delete a Kubernetes cluster NodeGroup

```
pharmer delete nodegroup [flags]
```

### Examples

```
pharmer delete nodegroup -k <cluster_name>
```

### Options

```
  -k, --cluster string   Name of the Kubernetes cluster
  -h, --help             help for nodegroup
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
* [pharmer delete](/docs/reference/pharmer_delete.md)	 - 

