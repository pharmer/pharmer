## pharmer edit cluster

Edit cluster object

### Synopsis


Edit cluster object

```
pharmer edit cluster [flags]
```

### Examples

```
pharmer edit cluster <cluster-name> --kubernetes-version=v1.8.0 
```

### Options

```
  -h, --help                        help for cluster
      --kubernetes-version string   Kubernetes version
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
* [pharmer edit](pharmer_edit.md)	 - 

