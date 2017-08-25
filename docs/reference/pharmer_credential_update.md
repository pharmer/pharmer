## pharmer credential update

Update an existing cloud credential

### Synopsis


Update an existing cloud credential

```
pharmer credential update [flags]
```

### Options

```
  -c, --credential string   Credential data
  -f, --file-path string    Credential file path
  -h, --help                help for update
  -n, --name string         Credential name
  -p, --provider string     Cloud provider name
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
* [pharmer credential](pharmer_credential.md)	 - Manage cloud provider credentials

