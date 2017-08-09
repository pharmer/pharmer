## pharmer reconfigure

Create/Resize/Upgrade/Downgrade a Kubernetes cluster instance group

### Synopsis


Create/Resize/Upgrade/Downgrade a Kubernetes cluster instance group

```
pharmer reconfigure [flags]
```

### Examples

```
appctl cluster reconfigure <name> --role=master|node --sku=n1-standard-1
```

### Options

```
      --apply-to-master   Set true to change version of master. Default set to false.
      --count int         Number of instances of this type (default -1)
  -h, --help              help for reconfigure
      --sku string        Instance type
      --version string    Kubernetes version
```

### Options inherited from parent commands

```
      --alsologtostderr                  log to standard error as well as files
      --log_backtrace_at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                   If non-empty, write log files in this directory
      --logtostderr                      log to standard error instead of files (default true)
      --stderrthreshold severity         logs at or above this threshold go to stderr (default 2)
  -v, --v Level                          log level for V logs
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging
```

### SEE ALSO
* [pharmer](pharmer.md)	 - Pharmer by Appscode - Manages farms

