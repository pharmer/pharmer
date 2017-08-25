## pharmer create

Create a Kubernetes cluster for a given cloud provider

### Synopsis


Create a Kubernetes cluster for a given cloud provider

```
pharmer create [flags]
```

### Examples

```
create --provider=(aws|gce|cc) --nodes=t1=1,t2=2 --zone=us-central1-f demo-cluster
```

### Options

```
      --credential-uid string   Use preconfigured cloud credential uid
      --do-not-delete           Set do not delete flag
      --gce-project gce         GCE project name(only applicable to gce provider)
  -h, --help                    help for create
      --nodes stringToInt       Node set configuration (default [])
      --provider string         Provider name
      --version string          Kubernetes version
      --zone string             Cloud provider zone name
```

### Options inherited from parent commands

```
      --alsologtostderr                  log to standard error as well as files
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
* [pharmer](pharmer.md)	 - Pharmer by Appscode - Manages farms

