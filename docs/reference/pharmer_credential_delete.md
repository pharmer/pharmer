## pharmer credential delete

Delete a cloud credential

### Synopsis


Delete a cloud credential

```
pharmer credential delete [flags]
```

### Examples

```
appctl credential delete --name="xyz"
```

### Options

```
  -h, --help          help for delete
  -n, --name string   credential name
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
* [pharmer credential](pharmer_credential.md)	 - Manage cloud provider credentials

