## pharmer create

Create a Kubernetes cluster for a given cloud provider

### Synopsis


Create a Kubernetes cluster for a given cloud provider

```
pharmer create [flags]
```

### Examples

```
create --provider=(aws|gce|cc) --nodes=t1:n1,t2:n2 --zone=us-central1-f demo-cluster
```

### Options

```
      --cloud-credential string   Use preconfigured cloud credential phid
      --do-not-delete             Set do not delete flag
      --gce-project gce           GCE project name(only applicable to gce provider)
  -h, --help                      help for create
      --nodes string              Node set configuration
      --provider string           Provider name
      --version string            Kubernetes version
      --zone string               Cloud provider zone name
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

