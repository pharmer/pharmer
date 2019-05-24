---
title: Pharmer Config
menu:
  product_pharmer_0.3.1:
    identifier: pharmer-config
    name: Pharmer Config
    parent: reference
product_name: pharmer
menu_name: product_pharmer_0.3.1
section_menu_id: reference
---
## pharmer config

Pharmer configuration

### Synopsis

Pharmer configuration

```
pharmer config [flags]
```

### Examples

```
pharmer config view
```

### Options

```
  -h, --help   help for config
```

### Options inherited from parent commands

```
      --alsologtostderr                  log to standard error as well as files
      --analytics                        Send analytical events to Google Guard (default true)
      --config-file string               Path to Pharmer config file
      --env string                       Environment used to enable debugging (default "prod")
      --kubeconfig string                Paths to a kubeconfig. Only required if out-of-cluster.
      --log_backtrace_at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                   If non-empty, write log files in this directory
      --logtostderr                      log to standard error instead of files (default true)
      --master --kubeconfig              (Deprecated: switch to --kubeconfig) The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.
      --stderrthreshold severity         logs at or above this threshold go to stderr (default 2)
  -v, --v Level                          log level for V logs
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging
```

### SEE ALSO

* [pharmer](/docs/reference/pharmer.md)	 - Pharmer by Appscode - Manages farms
* [pharmer config get-contexts](/docs/reference/pharmer_config_get-contexts.md)	 - List available contexts
* [pharmer config set-context](/docs/reference/pharmer_config_set-context.md)	 - Create  config object
* [pharmer config view](/docs/reference/pharmer_config_view.md)	 - Print Pharmer config

