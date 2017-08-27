## pharmer backup

Takes backup of YAML files of cluster

### Synopsis


Takes backup of YAML files of cluster

```
pharmer backup [flags]
```

### Options

```
      --backup-dir string   Directory where yaml files will be saved
      --cluster string      Name of cluster or Kube config context
  -h, --help                help for backup
      --sanitize             Sanitize fields in YAML
```

### Options inherited from parent commands

```
      --allow_verification_with_non_compliant_keys   Allow a SignatureVerifier to use keys which are technically non-compliant with RFC6962.
      --alsologtostderr                              log to standard error as well as files
      --config-file string                           Path to Pharmer config file
      --env string                                   Environment used to enable debugging (default "dev")
      --log_backtrace_at traceLocation               when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                               If non-empty, write log files in this directory
      --logtostderr                                  log to standard error instead of files (default true)
      --stderrthreshold severity                     logs at or above this threshold go to stderr (default 2)
  -v, --v Level                                      log level for V logs
      --vmodule moduleSpec                           comma-separated list of pattern=N settings for file-filtered logging
```

### SEE ALSO
* [pharmer](pharmer.md)	 - Pharmer by Appscode - Manages farms

